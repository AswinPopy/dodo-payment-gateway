package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
)

var (
	ErrWebhookURLRequired = errors.New("webhook url is required")
	ErrEventNotFound      = errors.New("event not found")
	ErrEventNotOwned      = errors.New("event does not belong to business")
)

type WebhookService struct {
	businessRepo      *repository.BusinessRepository
	outboundEventRepo *repository.OutboundEventRepository
}

func NewWebhookService(
	businessRepo *repository.BusinessRepository,
	outboundEventRepo *repository.OutboundEventRepository,
) *WebhookService {
	return &WebhookService{
		businessRepo:      businessRepo,
		outboundEventRepo: outboundEventRepo,
	}
}

func (s *WebhookService) SetEndpoint(
	ctx context.Context,
	businessID string,
	url string,
) (*model.Business, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, ErrWebhookURLRequired
	}

	business, err := s.businessRepo.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}

	secret := business.WebhookSecret
	if secret == "" {
		secret = webhook.NewSecret()
	}

	if err := s.businessRepo.UpdateWebhookEndpoint(
		ctx,
		businessID,
		url,
		secret,
	); err != nil {
		return nil, err
	}

	business.WebhookURL = &url
	business.WebhookSecret = secret

	return business, nil
}

func (s *WebhookService) ListEvents(
	ctx context.Context,
	businessID string,
	afterID string,
	limit int,
) ([]*model.OutboundEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.outboundEventRepo.ListByBusiness(ctx, businessID, afterID, limit)
}

func (s *WebhookService) GetEvent(
	ctx context.Context,
	businessID string,
	eventID string,
) (*model.OutboundEvent, error) {
	event, err := s.outboundEventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, ErrEventNotFound
	}

	if event.BusinessID != businessID {
		return nil, ErrEventNotOwned
	}

	return event, nil
}

func (s *WebhookService) Redeliver(
	ctx context.Context,
	businessID string,
	eventID string,
) (*model.OutboundEvent, error) {
	event, err := s.GetEvent(ctx, businessID, eventID)
	if err != nil {
		return nil, err
	}

	if err := s.outboundEventRepo.ResetForRedelivery(
		ctx,
		event.ID,
		time.Now(),
	); err != nil {
		return nil, err
	}

	return s.GetEvent(ctx, businessID, eventID)
}
