package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type CustomerRepository struct {
	db *pgxpool.Pool
}

func NewCustomerRepository(db *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{
		db: db,
	}
}

func (r *CustomerRepository) Create(
	ctx context.Context,
	customer *model.Customer,
) error {
	query := `
		INSERT INTO customers (
			id,
			business_id,
			name,
			email
		)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		customer.ID,
		customer.BusinessID,
		customer.Name,
		customer.Email,
	).Scan(&customer.CreatedAt)
}
