package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type APIKeyHandler struct {
	service *service.APIKeyService
}

func NewAPIKeyHandler(service *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

type createAPIKeyRequest struct {
	BusinessID string  `json:"business_id"`
	Name       *string `json:"name"`
}

type createAPIKeyResponse struct {
	ID         string  `json:"id"`
	BusinessID string  `json:"business_id"`
	Name       *string `json:"name,omitempty"`
	KeyPrefix  string  `json:"key_prefix"`
	APIKey     string  `json:"api_key"`
	CreatedAt  string  `json:"created_at"`
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req createAPIKeyRequest
	_ = c.ShouldBindJSON(&req)

	businessID, authenticated := c.Get(middleware.BusinessIDKey)
	businessIDString, ok := businessID.(string)

	if authenticated && ok {
		req.BusinessID = businessIDString
	}

	if req.BusinessID == "" {
		httperr.BadRequest(c, "business_id is required")
		return
	}

	rawKey, apiKey, err := h.service.CreateAPIKey(
		c.Request.Context(),
		req.BusinessID,
		req.Name,
		authenticated && ok,
	)
	if err != nil {
		writeAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createAPIKeyResponse{
		ID:         apiKey.ID,
		BusinessID: apiKey.BusinessID,
		Name:       apiKey.Name,
		KeyPrefix:  apiKey.KeyPrefix,
		APIKey:     rawKey,
		CreatedAt:  apiKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	businessID, ok := requireBusinessID(c)
	if !ok {
		return
	}

	keys, err := h.service.ListAPIKeys(c.Request.Context(), businessID)
	if err != nil {
		writeAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": keys,
	})
}

func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	businessID, ok := requireBusinessID(c)
	if !ok {
		return
	}

	key, err := h.service.RevokeAPIKey(
		c.Request.Context(),
		businessID,
		c.Param("id"),
	)
	if err != nil {
		writeAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusOK, key)
}

func (h *APIKeyHandler) RotateAPIKey(c *gin.Context) {
	businessID, ok := requireBusinessID(c)
	if !ok {
		return
	}

	rawKey, newKey, oldKey, err := h.service.RotateAPIKey(
		c.Request.Context(),
		businessID,
		c.Param("id"),
	)
	if err != nil {
		writeAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": rawKey,
		"created": createAPIKeyResponse{
			ID:         newKey.ID,
			BusinessID: newKey.BusinessID,
			Name:       newKey.Name,
			KeyPrefix:  newKey.KeyPrefix,
			APIKey:     rawKey,
			CreatedAt:  newKey.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		"revoked": oldKey,
	})
}

func requireBusinessID(c *gin.Context) (string, bool) {
	businessID, exists := c.Get(middleware.BusinessIDKey)
	if !exists {
		httperr.Unauthorized(c, "business context missing")
		return "", false
	}

	businessIDString, ok := businessID.(string)
	if !ok {
		httperr.Internal(c)
		return "", false
	}

	return businessIDString, true
}

func writeAPIKeyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBusinessNotFound),
		errors.Is(err, service.ErrAPIKeyNotFound):
		httperr.NotFound(c, err.Error())
	case errors.Is(err, service.ErrAPIKeyNotOwned):
		httperr.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrAPIKeyRevoked),
		errors.Is(err, service.ErrAPIKeyBootstrap):
		httperr.Conflict(c, err.Error())
	default:
		httperr.Internal(c)
	}
}
