package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
)

type chargeRequest struct {
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	CardToken      string `json:"card_token"`
	IdempotencyKey string `json:"idempotency_key"`
}

type chargeResponse struct {
	Status string `json:"status"`
	PSPRef string `json:"psp_ref,omitempty"`
	Code   string `json:"code,omitempty"`
}

type cacheEntry struct {
	ready    chan struct{}
	resp     chargeResponse
	httpCode int
}

type mockPSP struct {
	webhookURL    string
	webhookSecret string
	mu            sync.Mutex
	cache         map[string]*cacheEntry
}

func main() {
	addr := os.Getenv("MOCK_PSP_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	webhookURL := os.Getenv("INVOICE_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://localhost:8080/webhooks/psp"
	}

	webhookSecret := os.Getenv("PSP_WEBHOOK_SECRET")
	if webhookSecret == "" {
		webhookSecret = webhook.DefaultPSPSecret
	}

	psp := &mockPSP{
		webhookURL:    webhookURL,
		webhookSecret: webhookSecret,
		cache:         make(map[string]*cacheEntry),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/charges", psp.handleCharge)

	log.Printf("mock PSP listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (p *mockPSP) handleCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.CardToken == "" {
		http.Error(w, "card_token is required", http.StatusBadRequest)
		return
	}

	if req.IdempotencyKey != "" {
		entry, created := p.getOrStart(req.IdempotencyKey)
		if !created {
			<-entry.ready
			if entry.httpCode >= 500 {
				http.Error(w, "internal server error", entry.httpCode)
				return
			}
			writeJSON(w, entry.httpCode, entry.resp)
			return
		}

		p.completeCharge(w, req, entry)
		return
	}

	p.completeCharge(w, req, nil)
}

func (p *mockPSP) getOrStart(key string) (*cacheEntry, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok := p.cache[key]; ok {
		return entry, false
	}

	entry := &cacheEntry{
		ready:    make(chan struct{}),
		httpCode: http.StatusOK,
	}
	p.cache[key] = entry
	return entry, true
}

func (p *mockPSP) completeCharge(
	w http.ResponseWriter,
	req chargeRequest,
	entry *cacheEntry,
) {
	finish := func(code int, resp chargeResponse, notify bool) {
		if entry != nil {
			entry.httpCode = code
			entry.resp = resp
			close(entry.ready)
		}

		if notify && resp.Status == "succeeded" {
			go p.notifyWebhook("charge.succeeded", req, resp)
		}

		if code >= 500 {
			http.Error(w, "internal server error", code)
			return
		}

		writeJSON(w, code, resp)
	}

	switch req.CardToken {
	case "tok_success":
		time.Sleep(100 * time.Millisecond)
		finish(http.StatusOK, chargeResponse{
			Status: "succeeded",
			PSPRef: uuid.New().String(),
		}, true)

	case "tok_insufficient_funds":
		time.Sleep(100 * time.Millisecond)
		finish(http.StatusOK, chargeResponse{
			Status: "failed",
			Code:   "insufficient_funds",
		}, false)

	case "tok_card_declined":
		time.Sleep(100 * time.Millisecond)
		finish(http.StatusOK, chargeResponse{
			Status: "failed",
			Code:   "card_declined",
		}, false)

	case "tok_timeout":
		time.Sleep(30 * time.Second)
		finish(http.StatusOK, chargeResponse{
			Status: "succeeded",
			PSPRef: uuid.New().String(),
		}, true)

	case "tok_network_error":
		p.failUncached(entry, req.IdempotencyKey, http.StatusInternalServerError)
		http.Error(w, "internal server error", http.StatusInternalServerError)

	default:
		p.failUncached(entry, req.IdempotencyKey, http.StatusBadRequest)
		http.Error(w, "unknown card token", http.StatusBadRequest)
	}
}

func (p *mockPSP) failUncached(entry *cacheEntry, key string, code int) {
	if entry == nil {
		return
	}

	entry.httpCode = code
	p.mu.Lock()
	delete(p.cache, key)
	p.mu.Unlock()
	close(entry.ready)
}

func (p *mockPSP) notifyWebhook(
	eventType string,
	req chargeRequest,
	resp chargeResponse,
) {
	eventID := "evt_" + uuid.New().String()
	body, err := json.Marshal(map[string]any{
		"event_id":   eventID,
		"event_type": eventType,
		"data": map[string]any{
			"psp_ref":         resp.PSPRef,
			"idempotency_key": req.IdempotencyKey,
			"status":          resp.Status,
			"code":            resp.Code,
		},
	})
	if err != nil {
		log.Printf("webhook marshal failed: %v", err)
		return
	}

	client := &http.Client{Timeout: webhook.AttemptTimeout}
	delays := []time.Duration{0, 200 * time.Millisecond, time.Second}

	for i, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}

		now := time.Now()
		httpReq, err := http.NewRequest(
			http.MethodPost,
			p.webhookURL,
			bytes.NewReader(body),
		)
		if err != nil {
			log.Printf("webhook request failed: %v", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set(webhook.HeaderID, eventID)
		httpReq.Header.Set(webhook.HeaderTimestamp, fmt.Sprintf("%d", now.Unix()))
		httpReq.Header.Set(
			webhook.HeaderSignature,
			webhook.Sign(p.webhookSecret, eventID, now, body),
		)

		httpResp, err := client.Do(httpReq)
		if err == nil {
			httpResp.Body.Close()
			if httpResp.StatusCode < 500 {
				return
			}
		}

		if i == len(delays)-1 {
			log.Printf("webhook delivery failed for %s", req.IdempotencyKey)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, body chargeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
