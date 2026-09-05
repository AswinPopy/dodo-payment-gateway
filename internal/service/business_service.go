package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
)

type BusinessService struct {
	repo *repository.BusinessRepository
}

func NewBusinessService(repo *repository.BusinessRepository) *BusinessService {
	return &BusinessService{
		repo: repo,
	}
}

func (s *BusinessService) CreateBusiness(
	ctx context.Context,
	name string,
) (*model.Business, error) {
	business := &model.Business{
		ID:            uuid.New().String(),
		Name:          name,
		WebhookSecret: webhook.NewSecret(),
	}

	if err := s.repo.Create(ctx, business); err != nil {
		return nil, err
	}

	return business, nil
}
