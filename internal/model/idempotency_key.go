package model

import "time"

type IdempotencyKey struct {
	ID               string    `json:"id"`
	BusinessID       string    `json:"business_id"`
	IdempotencyKey   string    `json:"-"`
	InvoiceID        string    `json:"invoice_id"`
	PaymentAttemptID *string   `json:"payment_attempt_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
