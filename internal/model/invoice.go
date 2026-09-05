package model

import "time"

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "DRAFT"
	InvoiceStatusOpen          InvoiceStatus = "OPEN"
	InvoiceStatusPaid          InvoiceStatus = "PAID"
	InvoiceStatusVoid          InvoiceStatus = "VOID"
	InvoiceStatusUncollectible InvoiceStatus = "UNCOLLECTIBLE"
)

type Invoice struct {
	ID         string        `json:"id"`
	BusinessID string        `json:"business_id"`
	CustomerID string        `json:"customer_id"`
	Currency   string        `json:"currency"`
	Amount     int64         `json:"amount"`
	Status     InvoiceStatus `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}
