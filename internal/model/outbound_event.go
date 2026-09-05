package model

import (
	"encoding/json"
	"time"
)

type OutboundEventStatus string

const (
	OutboundEventStatusPending    OutboundEventStatus = "PENDING"
	OutboundEventStatusDelivering OutboundEventStatus = "DELIVERING"
	OutboundEventStatusDelivered  OutboundEventStatus = "DELIVERED"
	OutboundEventStatusFailed     OutboundEventStatus = "FAILED"
)

const (
	EventTypePaymentSucceeded = "payment.succeeded"
	EventTypePaymentFailed    = "payment.failed"
)

type OutboundEvent struct {
	ID           string              `json:"id"`
	BusinessID   string              `json:"business_id"`
	SourceID     string              `json:"source_id"`
	EventType    string              `json:"event_type"`
	Payload      json.RawMessage     `json:"payload"`
	Status       OutboundEventStatus `json:"status"`
	AttemptCount int                 `json:"attempt_count"`
	NextRetryAt  time.Time           `json:"next_retry_at"`
	LastError    *string             `json:"last_error,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	DeliveredAt  *time.Time          `json:"delivered_at,omitempty"`
}
