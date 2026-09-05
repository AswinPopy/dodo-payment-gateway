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
