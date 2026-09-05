package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/psp"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

type PaymentService struct {
	invoiceRepo        *repository.InvoiceRepository
	paymentAttemptRepo *repository.PaymentAttemptRepository
	idempotencyRepo    *repository.IdempotencyRepository
	psp                psp.PSP
}

func NewPaymentService(
	invoiceRepo *repository.InvoiceRepository,
	paymentAttemptRepo *repository.PaymentAttemptRepository,
	idempotencyRepo *repository.IdempotencyRepository,
	pspClient psp.PSP,
) *PaymentService {
	return &PaymentService{
		invoiceRepo:        invoiceRepo,
		paymentAttemptRepo: paymentAttemptRepo,
		idempotencyRepo:    idempotencyRepo,
		psp:                pspClient,
	}
}

func (s *PaymentService) PayInvoice(
	ctx context.Context,
	businessID string,
	invoiceID string,
	idempotencyKey string,
) (*model.PaymentAttempt, error) {

	existingKey, err := s.idempotencyRepo.GetByKey(
		ctx,
		businessID,
		idempotencyKey,
	)

	if err == nil {
		if existingKey.PaymentAttemptID == nil {
			return nil, errors.New("payment is already being processed")
		}

		return s.paymentAttemptRepo.GetByID(
			ctx,
			*existingKey.PaymentAttemptID,
		)
	}

	// 1. Get the invoice
	invoice, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}

	// 2. Make sure this invoice belongs to the authenticated business
	if invoice.BusinessID != businessID {
		return nil, errors.New("invoice does not belong to business")
	}

	// 3. Make sure the invoice can be paid
	if invoice.Status == model.InvoiceStatusPaid {
		return nil, errors.New("invoice already paid")
	}

	if invoice.Status == model.InvoiceStatusVoid {
		return nil, errors.New("invoice is void")
	}

	// 4. Move DRAFT → OPEN when payment starts
	if invoice.Status == model.InvoiceStatusDraft {
		if err := s.invoiceRepo.UpdateStatus(
			ctx,
			invoice.ID,
			model.InvoiceStatusOpen,
		); err != nil {
			return nil, err
		}
	}

	idempotencyRecord := &model.IdempotencyKey{
		ID:             uuid.New().String(),
		BusinessID:     businessID,
		IdempotencyKey: idempotencyKey,
		InvoiceID:      invoice.ID,
	}

	if err := s.idempotencyRepo.Create(ctx, idempotencyRecord); err != nil {
		var pgErr *pgconn.PgError

		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			return nil, err
		}

		existingKey, getErr := s.idempotencyRepo.GetByKey(
			ctx,
			businessID,
			idempotencyKey,
		)

		if getErr != nil {
			return nil, getErr
		}

		if existingKey.PaymentAttemptID == nil {
			return nil, errors.New("payment is already being processed")
		}

		return s.paymentAttemptRepo.GetByID(
			ctx,
			*existingKey.PaymentAttemptID,
		)
	}

	// 5. Create a payment attempt
	attempt := &model.PaymentAttempt{
		ID:         uuid.New().String(),
		InvoiceID:  invoice.ID,
		BusinessID: businessID,
		Amount:     invoice.Amount,
		Currency:   invoice.Currency,
		Status:     model.PaymentAttemptStatusPending,
	}

	if err := s.paymentAttemptRepo.Create(ctx, attempt); err != nil {
		return nil, err
	}

	if err := s.idempotencyRepo.SetPaymentAttemptID(
		ctx,
		idempotencyRecord.ID,
		attempt.ID,
	); err != nil {
		return nil, err
	}

	// 6. Send payment to PSP
	response, err := s.psp.Charge(ctx, psp.ChargeRequest{
		Amount:   invoice.Amount,
		Currency: invoice.Currency,
	})

	if err != nil {
		failureCode := err.Error()

		attempt.Status = model.PaymentAttemptStatusFailed
		attempt.FailureCode = &failureCode

		if updateErr := s.paymentAttemptRepo.UpdateStatus(
			ctx,
			attempt.ID,
			attempt.Status,
			nil,
			attempt.FailureCode,
		); updateErr != nil {
			return nil, updateErr
		}

		return attempt, nil
	}

	// 7. Payment succeeded
	attempt.Status = model.PaymentAttemptStatusSucceeded
	attempt.PSPTransactionID = &response.TransactionID

	if err := s.paymentAttemptRepo.UpdateStatus(
		ctx,
		attempt.ID,
		attempt.Status,
		attempt.PSPTransactionID,
		nil,
	); err != nil {
		return nil, err
	}

	if err := s.invoiceRepo.UpdateStatus(
		ctx,
		invoice.ID,
		model.InvoiceStatusPaid,
	); err != nil {
		return nil, err
	}

	return attempt, nil
}
