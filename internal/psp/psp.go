package psp

import "context"

type ChargeRequest struct {
	Amount   int64
	Currency string
}

type ChargeResponse struct {
	TransactionID string
}

type PSP interface {
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error)
}
