package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
)

type APIKeyService struct {
	apiKeyRepo   *repository.APIKeyRepository
	businessRepo *repository.BusinessRepository
}

func NewAPIKeyService(
	apiKeyRepo *repository.APIKeyRepository,
	businessRepo *repository.BusinessRepository,
) *APIKeyService {
	return &APIKeyService{
		apiKeyRepo:   apiKeyRepo,
		businessRepo: businessRepo,
	}
}

func (s *APIKeyService) CreateAPIKey(
	ctx context.Context,
	businessID string,
) (string, *model.APIKey, error) {

	exists, err := s.businessRepo.Exists(ctx, businessID)
	if err != nil {
		return "", nil, err
	}

	if !exists {
		return "", nil, errors.New("business not found")
	}

	rawBytes := make([]byte, 32)

	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, err
	}

	rawKey := "dodo_live_" + hex.EncodeToString(rawBytes)

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	apiKey := &model.APIKey{
		ID:         uuid.New().String(),
		BusinessID: businessID,
		KeyHash:    keyHash,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return "", nil, err
	}

	return rawKey, apiKey, nil
}
