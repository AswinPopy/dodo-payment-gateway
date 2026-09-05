package handler

import (
	"errors"
	"net/http"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	paymentService *service.PaymentService
	webhookService *service.WebhookService
}

func NewWebhookHandler(
	paymentService *service.PaymentService,
	webhookService *service.WebhookService,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService: paymentService,
		webhookService: webhookService,
	}
}

func (h *WebhookHandler) HandlePSPWebhook(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		httperr.BadRequest(c, "failed to read request body")
		return
	}

	err = h.paymentService.ProcessPSPWebhook(
		c.Request.Context(),
		payload,
		c.GetHeader(webhook.HeaderID),
		c.GetHeader(webhook.HeaderTimestamp),
		c.GetHeader(webhook.HeaderSignature),
	)
	if err != nil {
		switch {
		case errors.Is(err, webhook.ErrInvalidSignature),
			errors.Is(err, webhook.ErrReplay):
			httperr.Unauthorized(c, err.Error())
		case err.Error() == "invalid request body",
			err.Error() == "event_id is required",
			err.Error() == "event_type is required":
			httperr.BadRequest(c, err.Error())
		default:
			httperr.Internal(c)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"received": true,
	})
}

type setWebhookEndpointRequest struct {
	URL string `json:"url" binding:"required"`
}

func (h *WebhookHandler) SetEndpoint(c *gin.Context) {
	var req setWebhookEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, "url is required")
		return
	}

	businessID, ok := businessIDFromContext(c)
	if !ok {
		return
	}

	business, err := h.webhookService.SetEndpoint(
		c.Request.Context(),
		businessID,
		req.URL,
	)
	if err != nil {
		httperr.Internal(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"webhook_url":    business.WebhookURL,
		"webhook_secret": business.WebhookSecret,
	})
}

func (h *WebhookHandler) ListEvents(c *gin.Context) {
	businessID, ok := businessIDFromContext(c)
	if !ok {
		return
	}

	events, err := h.webhookService.ListEvents(
		c.Request.Context(),
		businessID,
		c.Query("after"),
		0,
	)
	if err != nil {
		httperr.Internal(c)
		return
	}

	if events == nil {
		events = []*model.OutboundEvent{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": events,
	})
}

func (h *WebhookHandler) GetEvent(c *gin.Context) {
	businessID, ok := businessIDFromContext(c)
	if !ok {
		return
	}

	event, err := h.webhookService.GetEvent(
		c.Request.Context(),
		businessID,
		c.Param("id"),
	)
	if err != nil {
		writeWebhookEventError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *WebhookHandler) RedeliverEvent(c *gin.Context) {
	businessID, ok := businessIDFromContext(c)
	if !ok {
		return
	}

	event, err := h.webhookService.Redeliver(
		c.Request.Context(),
		businessID,
		c.Param("id"),
	)
	if err != nil {
		writeWebhookEventError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, event)
}

func writeWebhookEventError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		httperr.NotFound(c, err.Error())
	case errors.Is(err, service.ErrEventNotOwned):
		httperr.Forbidden(c, err.Error())
	default:
		httperr.Internal(c)
	}
}

func businessIDFromContext(c *gin.Context) (string, bool) {
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
