package repository

import (
	"context"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type PaymentAttemptRepository struct {
	db DBTX
}

func NewPaymentAttemptRepository(db DBTX) *PaymentAttemptRepository {
	return &PaymentAttemptRepository{
		db: db,
	}
}

func (r *PaymentAttemptRepository) Create(
	ctx context.Context,
	attempt *model.PaymentAttempt,
) error {
	query := `
		INSERT INTO payment_attempts (
			id,
			invoice_id,
			business_id,
			amount,
			currency,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		attempt.ID,
		attempt.InvoiceID,
		attempt.BusinessID,
		attempt.Amount,
		attempt.Currency,
		attempt.Status,
	).Scan(
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	)
}

func (r *PaymentAttemptRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status model.PaymentAttemptStatus,
	pspTransactionID *string,
	failureCode *string,
) error {
	query := `
		UPDATE payment_attempts
		SET
			status = $1,
			psp_transaction_id = $2,
			failure_code = $3,
			updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.Exec(
		ctx,
		query,
		status,
		pspTransactionID,
		failureCode,
		id,
	)

	return err
}

func (r *PaymentAttemptRepository) UpdateStatusIfPending(
	ctx context.Context,
	id string,
	status model.PaymentAttemptStatus,
	pspTransactionID *string,
	failureCode *string,
) (bool, error) {
	query := `
		UPDATE payment_attempts
		SET
			status = $1,
			psp_transaction_id = $2,
			failure_code = $3,
			updated_at = NOW()
		WHERE id = $4
		  AND status = 'PENDING'
	`

	tag, err := r.db.Exec(
		ctx,
		query,
		status,
		pspTransactionID,
		failureCode,
		id,
	)
	if err != nil {
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

func (r *PaymentAttemptRepository) GetPendingByInvoiceID(
	ctx context.Context,
	invoiceID string,
) (*model.PaymentAttempt, error) {
	query := `
		SELECT
			id,
			invoice_id,
			business_id,
			amount,
			currency,
			status,
			psp_transaction_id,
			failure_code,
			created_at,
			updated_at
		FROM payment_attempts
		WHERE invoice_id = $1
		  AND status = 'PENDING'
	`

	attempt := &model.PaymentAttempt{}

	err := r.db.QueryRow(
		ctx,
		query,
		invoiceID,
	).Scan(
		&attempt.ID,
		&attempt.InvoiceID,
		&attempt.BusinessID,
		&attempt.Amount,
		&attempt.Currency,
		&attempt.Status,
		&attempt.PSPTransactionID,
		&attempt.FailureCode,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return attempt, nil
}

func (r *PaymentAttemptRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.PaymentAttempt, error) {
	query := `
		SELECT
			id,
			invoice_id,
			business_id,
			amount,
			currency,
			status,
			psp_transaction_id,
			failure_code,
			created_at,
			updated_at
		FROM payment_attempts
		WHERE id = $1
	`

	attempt := &model.PaymentAttempt{}

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&attempt.ID,
		&attempt.InvoiceID,
		&attempt.BusinessID,
		&attempt.Amount,
		&attempt.Currency,
		&attempt.Status,
		&attempt.PSPTransactionID,
		&attempt.FailureCode,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return attempt, nil
}
