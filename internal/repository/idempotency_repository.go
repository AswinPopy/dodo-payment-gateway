package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type IdempotencyRepository struct {
	db *pgxpool.Pool
}

func NewIdempotencyRepository(db *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{
		db: db,
	}
}

func (r *IdempotencyRepository) Create(
	ctx context.Context,
	key *model.IdempotencyKey,
) error {
	query := `
		INSERT INTO idempotency_keys (
			id,
			business_id,
			idempotency_key,
			invoice_id,
			payment_attempt_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		key.ID,
		key.BusinessID,
		key.IdempotencyKey,
		key.InvoiceID,
		key.PaymentAttemptID,
	).Scan(&key.CreatedAt)
}

func (r *IdempotencyRepository) GetByKey(
	ctx context.Context,
	businessID string,
	idempotencyKey string,
) (*model.IdempotencyKey, error) {
	query := `
		SELECT
			id,
			business_id,
			idempotency_key,
			invoice_id,
			payment_attempt_id,
			created_at
		FROM idempotency_keys
		WHERE business_id = $1
		  AND idempotency_key = $2
	`

	key := &model.IdempotencyKey{}

	err := r.db.QueryRow(
		ctx,
		query,
		businessID,
		idempotencyKey,
	).Scan(
		&key.ID,
		&key.BusinessID,
		&key.IdempotencyKey,
		&key.InvoiceID,
		&key.PaymentAttemptID,
		&key.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return key, nil
}

func (r *IdempotencyRepository) SetPaymentAttemptID(
	ctx context.Context,
	id string,
	paymentAttemptID string,
) error {
	query := `
		UPDATE idempotency_keys
		SET payment_attempt_id = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(
		ctx,
		query,
		paymentAttemptID,
		id,
	)

	return err
}
