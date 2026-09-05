package repository

import (
	"context"
	"time"

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
			name,
			key_prefix,
			key_hash
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		apiKey.ID,
		apiKey.BusinessID,
		apiKey.Name,
		apiKey.KeyPrefix,
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
			name,
			key_prefix,
			key_hash,
			created_at,
			last_used_at,
			revoked_at
		FROM api_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
	`

	return scanAPIKey(r.db.QueryRow(ctx, query, keyHash))
}

func (r *APIKeyRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.APIKey, error) {
	query := `
		SELECT
			id,
			business_id,
			name,
			key_prefix,
			key_hash,
			created_at,
			last_used_at,
			revoked_at
		FROM api_keys
		WHERE id = $1
	`

	return scanAPIKey(r.db.QueryRow(ctx, query, id))
}

func (r *APIKeyRepository) ListByBusiness(
	ctx context.Context,
	businessID string,
) ([]*model.APIKey, error) {
	query := `
		SELECT
			id,
			business_id,
			name,
			key_prefix,
			key_hash,
			created_at,
			last_used_at,
			revoked_at
		FROM api_keys
		WHERE business_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey

	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

func (r *APIKeyRepository) CountActive(
	ctx context.Context,
	businessID string,
) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM api_keys
		WHERE business_id = $1
		  AND revoked_at IS NULL
	`

	var count int
	err := r.db.QueryRow(ctx, query, businessID).Scan(&count)
	return count, err
}

func (r *APIKeyRepository) Revoke(
	ctx context.Context,
	id string,
	revokedAt time.Time,
) error {
	query := `
		UPDATE api_keys
		SET revoked_at = $1
		WHERE id = $2
		  AND revoked_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, revokedAt, id)
	return err
}

func (r *APIKeyRepository) TouchLastUsed(
	ctx context.Context,
	id string,
	usedAt time.Time,
) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, usedAt, id)
	return err
}

type apiKeyScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(row apiKeyScanner) (*model.APIKey, error) {
	apiKey := &model.APIKey{}

	err := row.Scan(
		&apiKey.ID,
		&apiKey.BusinessID,
		&apiKey.Name,
		&apiKey.KeyPrefix,
		&apiKey.KeyHash,
		&apiKey.CreatedAt,
		&apiKey.LastUsedAt,
		&apiKey.RevokedAt,
	)
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}
