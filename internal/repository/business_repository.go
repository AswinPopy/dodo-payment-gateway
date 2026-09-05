package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type BusinessRepository struct {
	db *pgxpool.Pool
}

func NewBusinessRepository(db *pgxpool.Pool) *BusinessRepository {
	return &BusinessRepository{
		db: db,
	}
}

func (r *BusinessRepository) Create(
	ctx context.Context,
	business *model.Business,
) error {
	query := `
		INSERT INTO businesses (id, name)
		VALUES ($1, $2)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		business.ID,
		business.Name,
	).Scan(&business.CreatedAt)
}

func (r *BusinessRepository) Exists(
	ctx context.Context,
	id string,
) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM businesses
			WHERE id = $1
		)
	`

	var exists bool

	err := r.db.QueryRow(ctx, query, id).Scan(&exists)

	return exists, err
}
