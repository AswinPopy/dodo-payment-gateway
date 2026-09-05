package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type InvoiceRepository struct {
	db *pgxpool.Pool
}

func NewInvoiceRepository(db *pgxpool.Pool) *InvoiceRepository {
	return &InvoiceRepository{
		db: db,
	}
}

func (r *InvoiceRepository) Create(
	ctx context.Context,
	invoice *model.Invoice,
) error {
	query := `
		INSERT INTO invoices (
			id,
			business_id,
			customer_id,
			currency,
			amount,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		invoice.ID,
		invoice.BusinessID,
		invoice.CustomerID,
		invoice.Currency,
		invoice.Amount,
		invoice.Status,
	).Scan(
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
	)
}

func (r *InvoiceRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.Invoice, error) {
	query := `
		SELECT
			id,
			business_id,
			customer_id,
			currency,
			amount,
			status,
			created_at,
			updated_at
		FROM invoices
		WHERE id = $1
	`

	invoice := &model.Invoice{}

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&invoice.ID,
		&invoice.BusinessID,
		&invoice.CustomerID,
		&invoice.Currency,
		&invoice.Amount,
		&invoice.Status,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return invoice, nil
}

func (r *InvoiceRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status model.InvoiceStatus,
) error {
	query := `
		UPDATE invoices
		SET
			status = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(
		ctx,
		query,
		status,
		id,
	)

	return err
}
