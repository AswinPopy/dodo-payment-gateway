package psp

import (
	"context"
	"errors"
)

var (
	// ErrTimeout means the PSP did not respond in time.
	// The charge may still complete on the PSP side.
	ErrTimeout = errors.New("psp timeout")

	// ErrUnavailable means the PSP returned 5xx or the
	// connection dropped. Outcome is unknown.
	ErrUnavailable = errors.New("psp unavailable")
)

type ChargeRequest struct {
	Amount         int64
	Currency       string
	CardToken      string
	IdempotencyKey string
}

type ChargeResponse struct {
	Status string
	PSPRef string
	Code   string
}

type PSP interface {
	Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error)
}

func IsUnknownOutcome(err error) bool {
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrUnavailable) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}
