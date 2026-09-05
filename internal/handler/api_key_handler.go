package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type APIKeyHandler struct {
	service *service.APIKeyService
}

func NewAPIKeyHandler(service *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

type createAPIKeyRequest struct {
	BusinessID string `json:"business_id" binding:"required"`
}

type createAPIKeyResponse struct {
	ID         string `json:"id"`
	BusinessID string `json:"business_id"`
	APIKey     string `json:"api_key"`
}

func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req createAPIKeyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "business_id is required",
		})
		return
	}

	rawKey, apiKey, err := h.service.CreateAPIKey(
		c.Request.Context(),
		req.BusinessID,
	)

	if err != nil {
		if err.Error() == "business not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "business not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create API key",
		})
		return
	}

	c.JSON(http.StatusCreated, createAPIKeyResponse{
		ID:         apiKey.ID,
		BusinessID: apiKey.BusinessID,
		APIKey:     rawKey,
	})
}
