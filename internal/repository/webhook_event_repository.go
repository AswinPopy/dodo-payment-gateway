package repository

import (
	"context"
	"time"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookEventRepository struct {
	db *pgxpool.Pool
}

func NewWebhookEventRepository(db *pgxpool.Pool) *WebhookEventRepository {
	return &WebhookEventRepository{db: db}
}

func (r *WebhookEventRepository) Create(
	ctx context.Context,
	event *model.WebhookEvent,
) error {
	query := `
		INSERT INTO webhook_events (
			id,
			event_id,
			event_type,
			payload,
			status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	return r.db.QueryRow(
		ctx,
		query,
		event.ID,
		event.EventID,
		event.EventType,
		event.Payload,
		event.Status,
	).Scan(&event.CreatedAt)
}

func (r *WebhookEventRepository) GetByEventID(
	ctx context.Context,
	eventID string,
) (*model.WebhookEvent, error) {
	query := `
		SELECT
			id,
			event_id,
			event_type,
			payload,
			status,
			created_at,
			processed_at
		FROM webhook_events
		WHERE event_id = $1
	`

	event := &model.WebhookEvent{}

	err := r.db.QueryRow(
		ctx,
		query,
		eventID,
	).Scan(
		&event.ID,
		&event.EventID,
		&event.EventType,
		&event.Payload,
		&event.Status,
		&event.CreatedAt,
		&event.ProcessedAt,
	)

	if err != nil {
		return nil, err
	}

	return event, nil
}

func (r *WebhookEventRepository) GetPending(
	ctx context.Context,
	limit int,
) ([]*model.WebhookEvent, error) {
	query := `
		SELECT
			id,
			event_id,
			event_type,
			payload,
			status,
			created_at,
			processed_at
		FROM webhook_events
		WHERE status = 'PENDING'
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.WebhookEvent

	for rows.Next() {
		event := &model.WebhookEvent{}

		err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.EventType,
			&event.Payload,
			&event.Status,
			&event.CreatedAt,
			&event.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *WebhookEventRepository) MarkProcessed(
	ctx context.Context,
	id string,
) error {
	query := `
		UPDATE webhook_events
		SET
			status = 'PROCESSED',
			processed_at = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(
		ctx,
		query,
		time.Now(),
		id,
	)

	return err
}

func (r *WebhookEventRepository) MarkFailed(
	ctx context.Context,
	id string,
) error {
	query := `
		UPDATE webhook_events
		SET status = 'FAILED'
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)

	return err
}
