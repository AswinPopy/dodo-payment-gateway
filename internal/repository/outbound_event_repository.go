package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
)

type OutboundEventRepository struct {
	db DBTX
}

func NewOutboundEventRepository(db DBTX) *OutboundEventRepository {
	return &OutboundEventRepository{db: db}
}

func (r *OutboundEventRepository) Create(
	ctx context.Context,
	event *model.OutboundEvent,
) error {
	query := `
		INSERT INTO outbound_events (
			id,
			business_id,
			source_id,
			event_type,
			payload,
			status,
			attempt_count,
			next_retry_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (event_type, source_id) DO NOTHING
		RETURNING created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		event.ID,
		event.BusinessID,
		event.SourceID,
		event.EventType,
		event.Payload,
		event.Status,
		event.AttemptCount,
		event.NextRetryAt,
	).Scan(&event.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	return err
}

func (r *OutboundEventRepository) GetByID(
	ctx context.Context,
	id string,
) (*model.OutboundEvent, error) {
	query := `
		SELECT
			id,
			business_id,
			source_id,
			event_type,
			payload,
			status,
			attempt_count,
			next_retry_at,
			last_error,
			created_at,
			delivered_at
		FROM outbound_events
		WHERE id = $1
	`

	return scanOutboundEvent(r.db.QueryRow(ctx, query, id))
}

func (r *OutboundEventRepository) ListByBusiness(
	ctx context.Context,
	businessID string,
	afterID string,
	limit int,
) ([]*model.OutboundEvent, error) {
	query := `
		SELECT
			id,
			business_id,
			source_id,
			event_type,
			payload,
			status,
			attempt_count,
			next_retry_at,
			last_error,
			created_at,
			delivered_at
		FROM outbound_events
		WHERE business_id = $1
		  AND ($2 = '' OR (created_at, id) > (
				SELECT created_at, id
				FROM outbound_events
				WHERE id = $2
			))
		ORDER BY created_at, id
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, businessID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.OutboundEvent

	for rows.Next() {
		event, err := scanOutboundEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *OutboundEventRepository) ClaimDue(
	ctx context.Context,
	limit int,
	now time.Time,
) ([]*model.OutboundEvent, error) {
	query := `
		UPDATE outbound_events
		SET status = 'DELIVERING'
		WHERE id IN (
			SELECT id
			FROM outbound_events
			WHERE status = 'PENDING'
			  AND next_retry_at <= $1
			ORDER BY next_retry_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING
			id,
			business_id,
			source_id,
			event_type,
			payload,
			status,
			attempt_count,
			next_retry_at,
			last_error,
			created_at,
			delivered_at
	`

	rows, err := r.db.Query(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.OutboundEvent

	for rows.Next() {
		event, err := scanOutboundEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *OutboundEventRepository) MarkDelivered(
	ctx context.Context,
	id string,
	deliveredAt time.Time,
) error {
	query := `
		UPDATE outbound_events
		SET
			status = 'DELIVERED',
			delivered_at = $1,
			last_error = NULL
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, deliveredAt, id)
	return err
}

func (r *OutboundEventRepository) MarkRetry(
	ctx context.Context,
	id string,
	attemptCount int,
	nextRetryAt time.Time,
	lastError string,
) error {
	query := `
		UPDATE outbound_events
		SET
			status = 'PENDING',
			attempt_count = $1,
			next_retry_at = $2,
			last_error = $3
		WHERE id = $4
	`

	_, err := r.db.Exec(ctx, query, attemptCount, nextRetryAt, lastError, id)
	return err
}

func (r *OutboundEventRepository) MarkFailed(
	ctx context.Context,
	id string,
	attemptCount int,
	lastError string,
) error {
	query := `
		UPDATE outbound_events
		SET
			status = 'FAILED',
			attempt_count = $1,
			last_error = $2
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, attemptCount, lastError, id)
	return err
}

func (r *OutboundEventRepository) ResetForRedelivery(
	ctx context.Context,
	id string,
	now time.Time,
) error {
	query := `
		UPDATE outbound_events
		SET
			status = 'PENDING',
			attempt_count = 0,
			next_retry_at = $1,
			last_error = NULL,
			delivered_at = NULL
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, now, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOutboundEvent(row rowScanner) (*model.OutboundEvent, error) {
	event := &model.OutboundEvent{}

	err := row.Scan(
		&event.ID,
		&event.BusinessID,
		&event.SourceID,
		&event.EventType,
		&event.Payload,
		&event.Status,
		&event.AttemptCount,
		&event.NextRetryAt,
		&event.LastError,
		&event.CreatedAt,
		&event.DeliveredAt,
	)
	if err != nil {
		return nil, err
	}

	return event, nil
}
