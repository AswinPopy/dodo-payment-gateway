package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type APIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(
	ctx context.Context,
	apiKey *model.APIKey,
) error {
	query := `
		INSERT INTO api_keys (
			id,
			business_id,
			key_hash
		)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		apiKey.ID,
		apiKey.BusinessID,
		apiKey.KeyHash,
	).Scan(&apiKey.CreatedAt)
}

func (r *APIKeyRepository) GetByHash(
	ctx context.Context,
	keyHash string,
) (*model.APIKey, error) {
	query := `
		SELECT
			id,
			business_id,
			key_hash,
			created_at,
			revoked_at
		FROM api_keys
		WHERE key_hash = $1
  			AND revoked_at IS NULL
	`

	apiKey := &model.APIKey{}

	err := r.db.QueryRow(
		ctx,
		query,
		keyHash,
	).Scan(
		&apiKey.ID,
		&apiKey.BusinessID,
		&apiKey.KeyHash,
		&apiKey.CreatedAt,
		&apiKey.RevokedAt,
	)

	if err != nil {
		return nil, err
	}

	return apiKey, nil
}
