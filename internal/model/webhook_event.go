package model

import "time"

type WebhookEventStatus string

const (
	WebhookEventStatusPending   WebhookEventStatus = "PENDING"
	WebhookEventStatusProcessed WebhookEventStatus = "PROCESSED"
	WebhookEventStatusFailed    WebhookEventStatus = "FAILED"
)

type WebhookEvent struct {
	ID          string             `json:"id"`
	EventID     string             `json:"event_id"`
	EventType   string             `json:"event_type"`
	Payload     []byte             `json:"payload"`
	Status      WebhookEventStatus `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	ProcessedAt *time.Time         `json:"processed_at,omitempty"`
}
