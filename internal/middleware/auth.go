package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
)

const (
	BusinessIDKey = "business_id"
	APIKeyIDKey   = "api_key_id"
)

func APIKeyAuth(apiKeyRepo *repository.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := authenticateAPIKey(c, apiKeyRepo, true)
		if !ok {
			return
		}

		c.Set(BusinessIDKey, apiKey.BusinessID)
		c.Set(APIKeyIDKey, apiKey.ID)

		go func(id string) {
			_ = apiKeyRepo.TouchLastUsed(
				context.Background(),
				id,
				time.Now(),
			)
		}(apiKey.ID)

		c.Next()
	}
}

func OptionalAPIKeyAuth(apiKeyRepo *repository.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.Next()
			return
		}

		apiKey, ok := authenticateAPIKey(c, apiKeyRepo, true)
		if !ok {
			return
		}

		c.Set(BusinessIDKey, apiKey.BusinessID)
		c.Set(APIKeyIDKey, apiKey.ID)
		c.Next()
	}
}

func authenticateAPIKey(
	c *gin.Context,
	apiKeyRepo *repository.APIKeyRepository,
	required bool,
) (*model.APIKey, bool) {
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		if !required {
			return nil, true
		}
		httperr.Unauthorized(c, "missing authorization header")
		c.Abort()
		return nil, false
	}

	parts := strings.SplitN(authHeader, " ", 2)

	if len(parts) != 2 || parts[0] != "Bearer" {
		httperr.Unauthorized(c, "invalid authorization header")
		c.Abort()
		return nil, false
	}

	hash := sha256.Sum256([]byte(parts[1]))
	keyHash := hex.EncodeToString(hash[:])

	apiKey, err := apiKeyRepo.GetByHash(
		c.Request.Context(),
		keyHash,
	)
	if err != nil {
		httperr.Unauthorized(c, "invalid API key")
		c.Abort()
		return nil, false
	}

	return apiKey, true
}
