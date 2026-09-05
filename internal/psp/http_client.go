package psp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const defaultTimeout = 2 * time.Second

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			// Shorter than tok_timeout's 30s sleep so the
			// invoice API never hangs waiting on the PSP.
			Timeout: defaultTimeout,
		},
	}
}

type chargeHTTPRequest struct {
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	CardToken      string `json:"card_token"`
	IdempotencyKey string `json:"idempotency_key"`
}

type chargeHTTPResponse struct {
	Status string `json:"status"`
	PSPRef string `json:"psp_ref"`
	Code   string `json:"code"`
}

func (c *HTTPClient) Charge(
	ctx context.Context,
	req ChargeRequest,
) (*ChargeResponse, error) {
	payload, err := json.Marshal(chargeHTTPRequest{
		Amount:         req.Amount,
		Currency:       req.Currency,
		CardToken:      req.CardToken,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/charges",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if isTimeout(err) {
			return nil, ErrTimeout
		}
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, ErrUnavailable
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("psp rejected request: status %d", resp.StatusCode)
	}

	var body chargeHTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: invalid response body", ErrUnavailable)
	}

	return &ChargeResponse{
		Status: body.Status,
		PSPRef: body.PSPRef,
		Code:   body.Code,
	}, nil
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
