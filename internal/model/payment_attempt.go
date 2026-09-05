package model

import "time"

type PaymentAttemptStatus string

const (
	PaymentAttemptStatusPending   PaymentAttemptStatus = "PENDING"
	PaymentAttemptStatusSucceeded PaymentAttemptStatus = "SUCCEEDED"
	PaymentAttemptStatusFailed    PaymentAttemptStatus = "FAILED"
)

type PaymentAttempt struct {
	ID               string               `json:"id"`
	InvoiceID        string               `json:"invoice_id"`
	BusinessID       string               `json:"business_id"`
	Amount           int64                `json:"amount"`
	Currency         string               `json:"currency"`
	Status           PaymentAttemptStatus `json:"status"`
	PSPTransactionID *string              `json:"psp_transaction_id,omitempty"`
	FailureCode      *string              `json:"failure_code,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}
