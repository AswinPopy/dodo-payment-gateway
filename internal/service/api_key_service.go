package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
)

const apiKeySecretPrefix = "dodo_live_"

var (
	ErrAPIKeyNotFound   = errors.New("api key not found")
	ErrAPIKeyNotOwned   = errors.New("api key does not belong to business")
	ErrAPIKeyRevoked    = errors.New("api key already revoked")
	ErrAPIKeyBootstrap  = errors.New("business already has an active api key")
	ErrBusinessNotFound = errors.New("business not found")
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
	name *string,
	authenticated bool,
) (string, *model.APIKey, error) {
	exists, err := s.businessRepo.Exists(ctx, businessID)
	if err != nil {
		return "", nil, err
	}
	if !exists {
		return "", nil, ErrBusinessNotFound
	}

	// Unauthenticated create is only for the first key
	// (bootstrap). After that, a leaked create endpoint
	// would mint keys for any business.
	if !authenticated {
		active, err := s.apiKeyRepo.CountActive(ctx, businessID)
		if err != nil {
			return "", nil, err
		}
		if active > 0 {
			return "", nil, ErrAPIKeyBootstrap
		}
	}

	return s.issueKey(ctx, businessID, name)
}

func (s *APIKeyService) ListAPIKeys(
	ctx context.Context,
	businessID string,
) ([]*model.APIKey, error) {
	keys, err := s.apiKeyRepo.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []*model.APIKey{}
	}
	return keys, nil
}

func (s *APIKeyService) RevokeAPIKey(
	ctx context.Context,
	businessID string,
	keyID string,
) (*model.APIKey, error) {
	key, err := s.ownedKey(ctx, businessID, keyID)
	if err != nil {
		return nil, err
	}
	if key.RevokedAt != nil {
		return nil, ErrAPIKeyRevoked
	}

	now := time.Now()
	if err := s.apiKeyRepo.Revoke(ctx, key.ID, now); err != nil {
		return nil, err
	}

	key.RevokedAt = &now
	return key, nil
}

func (s *APIKeyService) RotateAPIKey(
	ctx context.Context,
	businessID string,
	keyID string,
) (string, *model.APIKey, *model.APIKey, error) {
	oldKey, err := s.ownedKey(ctx, businessID, keyID)
	if err != nil {
		return "", nil, nil, err
	}
	if oldKey.RevokedAt != nil {
		return "", nil, nil, ErrAPIKeyRevoked
	}

	rawKey, newKey, err := s.issueKey(ctx, businessID, oldKey.Name)
	if err != nil {
		return "", nil, nil, err
	}

	now := time.Now()
	if err := s.apiKeyRepo.Revoke(ctx, oldKey.ID, now); err != nil {
		return "", nil, nil, err
	}

	oldKey.RevokedAt = &now
	return rawKey, newKey, oldKey, nil
}

func (s *APIKeyService) ownedKey(
	ctx context.Context,
	businessID string,
	keyID string,
) (*model.APIKey, error) {
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.BusinessID != businessID {
		return nil, ErrAPIKeyNotOwned
	}
	return key, nil
}

func (s *APIKeyService) issueKey(
	ctx context.Context,
	businessID string,
	name *string,
) (string, *model.APIKey, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, err
	}

	secret := hex.EncodeToString(rawBytes)
	rawKey := apiKeySecretPrefix + secret

	hash := sha256.Sum256([]byte(rawKey))

	apiKey := &model.APIKey{
		ID:         uuid.New().String(),
		BusinessID: businessID,
		Name:       name,
		KeyPrefix:  apiKeySecretPrefix + secret[:8],
		KeyHash:    hex.EncodeToString(hash[:]),
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return "", nil, err
	}

	return rawKey, apiKey, nil
}
