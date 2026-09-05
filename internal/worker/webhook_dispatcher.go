package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
)

type WebhookDispatcher struct {
	businessRepo      *repository.BusinessRepository
	outboundEventRepo *repository.OutboundEventRepository
	httpClient        *http.Client
}

func NewWebhookDispatcher(
	businessRepo *repository.BusinessRepository,
	outboundEventRepo *repository.OutboundEventRepository,
) *WebhookDispatcher {
	return &WebhookDispatcher{
		businessRepo:      businessRepo,
		outboundEventRepo: outboundEventRepo,
		httpClient: &http.Client{
			Timeout: webhook.AttemptTimeout,
		},
	}
}

func (d *WebhookDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(webhook.DispatcherPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.DeliverDue(ctx); err != nil {
				log.Printf("webhook dispatcher: %v", err)
			}
		}
	}
}

func (d *WebhookDispatcher) DeliverDue(ctx context.Context) error {
	events, err := d.outboundEventRepo.ClaimDue(
		ctx,
		webhook.ClaimBatchSize,
		time.Now(),
	)
	if err != nil {
		return err
	}

	for _, event := range events {
		d.deliverOne(ctx, event)
	}

	return nil
}

func (d *WebhookDispatcher) deliverOne(
	ctx context.Context,
	event *model.OutboundEvent,
) {
	business, err := d.businessRepo.GetByID(ctx, event.BusinessID)
	if err != nil {
		d.recordFailure(ctx, event, "business not found")
		return
	}

	if business.WebhookURL == nil || *business.WebhookURL == "" {
		d.recordFailure(ctx, event, "webhook url not configured")
		return
	}

	now := time.Now()
	signature := webhook.Sign(
		business.WebhookSecret,
		event.ID,
		now,
		event.Payload,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		*business.WebhookURL,
		bytes.NewReader(event.Payload),
	)
	if err != nil {
		d.recordFailure(ctx, event, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(webhook.HeaderID, event.ID)
	req.Header.Set(webhook.HeaderTimestamp, fmt.Sprintf("%d", now.Unix()))
	req.Header.Set(webhook.HeaderSignature, signature)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.recordFailure(ctx, event, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := d.outboundEventRepo.MarkDelivered(ctx, event.ID, time.Now()); err != nil {
			log.Printf("webhook dispatcher: mark delivered: %v", err)
		}
		return
	}

	d.recordFailure(ctx, event, fmt.Sprintf("http %d", resp.StatusCode))
}

func (d *WebhookDispatcher) recordFailure(
	ctx context.Context,
	event *model.OutboundEvent,
	reason string,
) {
	failedAttempts := event.AttemptCount + 1
	next, ok := webhook.NextRetryAt(failedAttempts, time.Now())
	if !ok {
		if err := d.outboundEventRepo.MarkFailed(
			ctx,
			event.ID,
			failedAttempts,
			reason,
		); err != nil {
			log.Printf("webhook dispatcher: mark failed: %v", err)
		}
		return
	}

	if err := d.outboundEventRepo.MarkRetry(
		ctx,
		event.ID,
		failedAttempts,
		next,
		reason,
	); err != nil {
		log.Printf("webhook dispatcher: mark retry: %v", err)
	}
}
