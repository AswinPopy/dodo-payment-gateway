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
		INSERT INTO businesses (id, name, webhook_secret)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		business.ID,
		business.Name,
		business.WebhookSecret,
	).Scan(&business.CreatedAt)
}

func (r *BusinessRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.Business, error) {
	query := `
		SELECT
			id,
			name,
			webhook_url,
			webhook_secret,
			created_at
		FROM businesses
		WHERE id = $1
	`

	business := &model.Business{}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&business.ID,
		&business.Name,
		&business.WebhookURL,
		&business.WebhookSecret,
		&business.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return business, nil
}

func (r *BusinessRepository) UpdateWebhookEndpoint(
	ctx context.Context,
	id string,
	url string,
	secret string,
) error {
	query := `
		UPDATE businesses
		SET
			webhook_url = $1,
			webhook_secret = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, url, secret, id)
	return err
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
