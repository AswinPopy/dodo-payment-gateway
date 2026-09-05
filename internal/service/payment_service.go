package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/psp"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
)

var (
	ErrInvoiceNotFound        = errors.New("invoice not found")
	ErrInvoiceNotOwned        = errors.New("invoice does not belong to business")
	ErrInvoiceAlreadyPaid     = errors.New("invoice already paid")
	ErrInvoiceVoid            = errors.New("invoice is void")
	ErrInvoiceUncollectible   = errors.New("invoice is uncollectible")
	ErrPaymentInProgress      = errors.New("payment is already being processed")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrPaymentAttemptNotFound = errors.New("payment attempt not found")
	ErrPaymentAttemptNotOwned = errors.New("payment attempt does not belong to business")
)

type PaymentService struct {
	db                 *pgxpool.Pool
	invoiceRepo        *repository.InvoiceRepository
	paymentAttemptRepo *repository.PaymentAttemptRepository
	idempotencyRepo    *repository.IdempotencyRepository
	webhookEventRepo   *repository.WebhookEventRepository
	outboundEventRepo  *repository.OutboundEventRepository
	pspWebhookSecret   string
	psp                psp.PSP
}

func NewPaymentService(
	db *pgxpool.Pool,
	invoiceRepo *repository.InvoiceRepository,
	paymentAttemptRepo *repository.PaymentAttemptRepository,
	idempotencyRepo *repository.IdempotencyRepository,
	webhookEventRepo *repository.WebhookEventRepository,
	outboundEventRepo *repository.OutboundEventRepository,
	pspWebhookSecret string,
	pspClient psp.PSP,
) *PaymentService {
	return &PaymentService{
		db:                 db,
		invoiceRepo:        invoiceRepo,
		paymentAttemptRepo: paymentAttemptRepo,
		idempotencyRepo:    idempotencyRepo,
		webhookEventRepo:   webhookEventRepo,
		outboundEventRepo:  outboundEventRepo,
		pspWebhookSecret:   pspWebhookSecret,
		psp:                pspClient,
	}
}

func (s *PaymentService) PayInvoice(
	ctx context.Context,
	businessID string,
	invoiceID string,
	idempotencyKey string,
	cardToken string,
) (*model.PaymentAttempt, error) {

	requestHash := hashPayRequest(invoiceID, cardToken)

	existingKey, err := s.idempotencyRepo.GetByKey(
		ctx,
		businessID,
		idempotencyKey,
	)
	if err == nil {
		return s.replayIdempotentPay(
			ctx,
			existingKey,
			invoiceID,
			cardToken,
			requestHash,
		)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	attempt, err := s.createPendingAttempt(
		ctx,
		businessID,
		invoiceID,
		idempotencyKey,
		requestHash,
	)
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) ||
			errors.Is(err, ErrPaymentInProgress) ||
			errors.Is(err, ErrInvoiceAlreadyPaid) ||
			errors.Is(err, ErrInvoiceVoid) ||
			errors.Is(err, ErrInvoiceNotOwned) ||
			errors.Is(err, ErrInvoiceNotFound) {
			return nil, err
		}

		if isUniqueViolation(err) {
			existingKey, getErr := s.idempotencyRepo.GetByKey(
				ctx,
				businessID,
				idempotencyKey,
			)
			if getErr != nil {
				return nil, getErr
			}

			return s.replayIdempotentPay(
				ctx,
				existingKey,
				invoiceID,
				cardToken,
				requestHash,
			)
		}

		return nil, err
	}

	return s.chargeAndSettle(ctx, attempt, cardToken)
}

func (s *PaymentService) GetPaymentAttempt(
	ctx context.Context,
	businessID string,
	attemptID string,
) (*model.PaymentAttempt, error) {
	attempt, err := s.paymentAttemptRepo.GetByID(ctx, attemptID)
	if err != nil {
		return nil, ErrPaymentAttemptNotFound
	}

	if attempt.BusinessID != businessID {
		return nil, ErrPaymentAttemptNotOwned
	}

	return attempt, nil
}

type pspWebhookPayload struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Data      struct {
		PSPRef         string `json:"psp_ref"`
		IdempotencyKey string `json:"idempotency_key"`
		Status         string `json:"status"`
		Code           string `json:"code"`
	} `json:"data"`
}

func (s *PaymentService) ProcessPSPWebhook(
	ctx context.Context,
	payload []byte,
	eventID string,
	timestamp string,
	signature string,
) error {
	if err := webhook.Verify(
		s.pspWebhookSecret,
		eventID,
		timestamp,
		signature,
		payload,
		time.Now(),
	); err != nil {
		return err
	}

	var body pspWebhookPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return errors.New("invalid request body")
	}

	if body.EventID == "" {
		return errors.New("event_id is required")
	}

	if body.EventType == "" {
		return errors.New("event_type is required")
	}

	event := &model.WebhookEvent{
		ID:        uuid.New().String(),
		EventID:   body.EventID,
		EventType: body.EventType,
		Payload:   payload,
		Status:    model.WebhookEventStatusPending,
	}

	if err := s.webhookEventRepo.Create(ctx, event); err != nil {
		if !isUniqueViolation(err) {
			return err
		}

		existing, getErr := s.webhookEventRepo.GetByEventID(ctx, body.EventID)
		if getErr != nil {
			return getErr
		}

		if existing.Status == model.WebhookEventStatusProcessed {
			return nil
		}

		event = existing
	}

	if err := s.applyWebhook(ctx, body); err != nil {
		_ = s.webhookEventRepo.MarkFailed(ctx, event.ID)
		return err
	}

	return s.webhookEventRepo.MarkProcessed(ctx, event.ID)
}

func (s *PaymentService) applyWebhook(
	ctx context.Context,
	body pspWebhookPayload,
) error {
	if body.Data.IdempotencyKey == "" {
		return nil
	}

	attempt, err := s.paymentAttemptRepo.GetByID(ctx, body.Data.IdempotencyKey)
	if err != nil {
		return nil
	}

	if body.EventType == "charge.succeeded" || body.Data.Status == "succeeded" {
		return s.persistSuccess(ctx, attempt, body.Data.PSPRef)
	}

	if body.EventType == "charge.failed" || body.Data.Status == "failed" {
		code := body.Data.Code
		if code == "" {
			code = "card_declined"
		}

		return s.persistFailure(ctx, attempt, code)
	}

	return nil
}

func (s *PaymentService) replayIdempotentPay(
	ctx context.Context,
	existingKey *model.IdempotencyKey,
	invoiceID string,
	cardToken string,
	requestHash string,
) (*model.PaymentAttempt, error) {
	if existingKey.RequestHash != requestHash || existingKey.InvoiceID != invoiceID {
		return nil, ErrIdempotencyConflict
	}

	if existingKey.PaymentAttemptID == nil {
		return nil, ErrPaymentInProgress
	}

	attempt, err := s.paymentAttemptRepo.GetByID(
		ctx,
		*existingKey.PaymentAttemptID,
	)
	if err != nil {
		return nil, err
	}

	if attempt.Status != model.PaymentAttemptStatusPending {
		return attempt, nil
	}

	return s.chargeAndSettle(ctx, attempt, cardToken)
}

func (s *PaymentService) createPendingAttempt(
	ctx context.Context,
	businessID string,
	invoiceID string,
	idempotencyKey string,
	requestHash string,
) (*model.PaymentAttempt, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	invoiceRepoTx := repository.NewInvoiceRepository(tx)
	attemptRepoTx := repository.NewPaymentAttemptRepository(tx)
	idempotencyRepoTx := repository.NewIdempotencyRepository(tx)

	// Row-level lock serializes concurrent POSTs against
	// the same invoice. The unique PENDING index is the
	// safety net if a second writer still races.
	invoice, err := invoiceRepoTx.GetByIDForUpdate(ctx, invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}

	if invoice.BusinessID != businessID {
		return nil, ErrInvoiceNotOwned
	}

	if err := payableInvoice(invoice.Status); err != nil {
		return nil, err
	}

	if invoice.Status == model.InvoiceStatusDraft {
		if err := invoiceRepoTx.UpdateStatus(
			ctx,
			invoice.ID,
			model.InvoiceStatusOpen,
		); err != nil {
			return nil, err
		}
	}

	attempt := &model.PaymentAttempt{
		ID:         uuid.New().String(),
		InvoiceID:  invoice.ID,
		BusinessID: businessID,
		Amount:     invoice.Amount,
		Currency:   invoice.Currency,
		Status:     model.PaymentAttemptStatusPending,
	}

	idempotencyRecord := &model.IdempotencyKey{
		ID:               uuid.New().String(),
		BusinessID:       businessID,
		IdempotencyKey:   idempotencyKey,
		InvoiceID:        invoice.ID,
		RequestHash:      requestHash,
		PaymentAttemptID: &attempt.ID,
	}

	// Insert the key first so a same-key racer hits 23505
	// and replays, instead of being treated as a second payer.
	if err := idempotencyRepoTx.Create(ctx, idempotencyRecord); err != nil {
		return nil, err
	}

	_, pendingErr := attemptRepoTx.GetPendingByInvoiceID(ctx, invoice.ID)
	if pendingErr == nil {
		return nil, ErrPaymentInProgress
	}
	if !errors.Is(pendingErr, pgx.ErrNoRows) {
		return nil, pendingErr
	}

	if err := attemptRepoTx.Create(ctx, attempt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPaymentInProgress
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return attempt, nil
}

func (s *PaymentService) chargeAndSettle(
	ctx context.Context,
	attempt *model.PaymentAttempt,
	cardToken string,
) (*model.PaymentAttempt, error) {
	response, err := s.psp.Charge(ctx, psp.ChargeRequest{
		Amount:         attempt.Amount,
		Currency:       attempt.Currency,
		CardToken:      cardToken,
		IdempotencyKey: attempt.ID,
	})

	if err != nil {
		// tok_timeout / tok_network_error: outcome is
		// unknown. Leave PENDING. A later retry uses the
		// same PSP idempotency key, or a webhook settles it.
		if psp.IsUnknownOutcome(err) {
			slog.Info("psp charge unknown",
				"attempt_id", attempt.ID,
				"invoice_id", attempt.InvoiceID,
				"reason", err.Error(),
			)
			return attempt, nil
		}

		if persistErr := s.persistFailure(ctx, attempt, err.Error()); persistErr != nil {
			return nil, persistErr
		}

		slog.Info("psp charge failed",
			"attempt_id", attempt.ID,
			"invoice_id", attempt.InvoiceID,
		)
		return attempt, nil
	}

	if response.Status != "succeeded" {
		code := response.Code
		if code == "" {
			code = "card_declined"
		}

		if persistErr := s.persistFailure(ctx, attempt, code); persistErr != nil {
			return nil, persistErr
		}

		slog.Info("psp charge declined",
			"attempt_id", attempt.ID,
			"invoice_id", attempt.InvoiceID,
			"code", code,
		)
		return attempt, nil
	}

	if err := s.persistSuccess(ctx, attempt, response.PSPRef); err != nil {
		return nil, err
	}

	slog.Info("psp charge succeeded",
		"attempt_id", attempt.ID,
		"invoice_id", attempt.InvoiceID,
	)

	return attempt, nil
}

func (s *PaymentService) persistSuccess(
	ctx context.Context,
	attempt *model.PaymentAttempt,
	pspRef string,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	invoiceRepoTx := repository.NewInvoiceRepository(tx)
	attemptRepoTx := repository.NewPaymentAttemptRepository(tx)

	if _, err := invoiceRepoTx.GetByIDForUpdate(ctx, attempt.InvoiceID); err != nil {
		return err
	}

	updated, err := attemptRepoTx.UpdateStatusIfPending(
		ctx,
		attempt.ID,
		model.PaymentAttemptStatusSucceeded,
		&pspRef,
		nil,
	)
	if err != nil {
		return err
	}
	if !updated {
		existing, getErr := attemptRepoTx.GetByID(ctx, attempt.ID)
		if getErr != nil {
			return getErr
		}
		*attempt = *existing
		if existing.Status != model.PaymentAttemptStatusSucceeded {
			return tx.Commit(ctx)
		}
	}

	if err := invoiceRepoTx.MarkPaidIfOpen(ctx, attempt.InvoiceID); err != nil {
		return err
	}

	attempt.Status = model.PaymentAttemptStatusSucceeded
	attempt.PSPTransactionID = &pspRef
	attempt.FailureCode = nil

	if err := enqueueOutboundEvent(
		ctx,
		repository.NewOutboundEventRepository(tx),
		attempt,
		model.EventTypePaymentSucceeded,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	attempt.Status = model.PaymentAttemptStatusSucceeded
	attempt.PSPTransactionID = &pspRef
	attempt.FailureCode = nil

	return nil
}

func (s *PaymentService) persistFailure(
	ctx context.Context,
	attempt *model.PaymentAttempt,
	code string,
) error {
	updated, err := s.paymentAttemptRepo.UpdateStatusIfPending(
		ctx,
		attempt.ID,
		model.PaymentAttemptStatusFailed,
		nil,
		&code,
	)
	if err != nil {
		return err
	}
	if !updated {
		existing, getErr := s.paymentAttemptRepo.GetByID(ctx, attempt.ID)
		if getErr != nil {
			return getErr
		}
		*attempt = *existing
		return nil
	}

	attempt.Status = model.PaymentAttemptStatusFailed
	attempt.FailureCode = &code

	return enqueueOutboundEvent(
		ctx,
		s.outboundEventRepo,
		attempt,
		model.EventTypePaymentFailed,
	)
}

func enqueueOutboundEvent(
	ctx context.Context,
	repo *repository.OutboundEventRepository,
	attempt *model.PaymentAttempt,
	eventType string,
) error {
	payload, err := json.Marshal(map[string]any{
		"id":   attempt.ID,
		"type": eventType,
		"data": map[string]any{
			"payment_attempt_id": attempt.ID,
			"invoice_id":         attempt.InvoiceID,
			"business_id":        attempt.BusinessID,
			"amount":             attempt.Amount,
			"currency":           attempt.Currency,
			"status":             attempt.Status,
			"psp_transaction_id": attempt.PSPTransactionID,
			"failure_code":       attempt.FailureCode,
		},
	})
	if err != nil {
		return err
	}

	event := &model.OutboundEvent{
		ID:           uuid.New().String(),
		BusinessID:   attempt.BusinessID,
		SourceID:     attempt.ID + ":" + eventType,
		EventType:    eventType,
		Payload:      payload,
		Status:       model.OutboundEventStatusPending,
		AttemptCount: 0,
		NextRetryAt:  time.Now(),
	}

	return repo.Create(ctx, event)
}

func payableInvoice(status model.InvoiceStatus) error {
	switch status {
	case model.InvoiceStatusDraft, model.InvoiceStatusOpen:
		return nil
	case model.InvoiceStatusPaid:
		return ErrInvoiceAlreadyPaid
	case model.InvoiceStatusVoid:
		return ErrInvoiceVoid
	case model.InvoiceStatusUncollectible:
		return ErrInvoiceUncollectible
	default:
		return ErrInvoiceVoid
	}
}

func hashPayRequest(invoiceID string, cardToken string) string {
	sum := sha256.Sum256([]byte(invoiceID + "\n" + cardToken))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
