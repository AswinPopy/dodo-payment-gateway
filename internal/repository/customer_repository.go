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

func (r *CustomerRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.Customer, error) {
	query := `
		SELECT
			id,
			business_id,
			name,
			email,
			created_at
		FROM customers
		WHERE id = $1
	`

	customer := &model.Customer{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&customer.ID,
		&customer.BusinessID,
		&customer.Name,
		&customer.Email,
		&customer.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return customer, nil
}
