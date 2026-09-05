package psp

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type MockPSP struct {
	ShouldFail bool
}

func NewMockPSP() *MockPSP {
	return &MockPSP{}
}

func (m *MockPSP) Charge(
	ctx context.Context,
	req ChargeRequest,
) (*ChargeResponse, error) {

	if req.Amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	if req.Currency == "" {
		return nil, errors.New("invalid currency")
	}

	if m.ShouldFail {
		return nil, errors.New("card_declined")
	}

	return &ChargeResponse{
		TransactionID: "psp_" + uuid.New().String(),
	}, nil
}
