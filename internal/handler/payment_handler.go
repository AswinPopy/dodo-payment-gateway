package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service: service,
	}
}

type payInvoiceRequest struct {
	CardToken string `json:"card_token" binding:"required"`
}

func (h *PaymentHandler) PayInvoice(c *gin.Context) {
	invoiceID := c.Param("id")

	if invoiceID == "" {
		httperr.BadRequest(c, "invoice id is required")
		return
	}

	var req payInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, "card_token is required")
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	if idempotencyKey == "" {
		httperr.BadRequest(c, "Idempotency-Key header is required")
		return
	}

	businessID, exists := c.Get(middleware.BusinessIDKey)
	if !exists {
		httperr.Unauthorized(c, "business context missing")
		return
	}

	businessIDString, ok := businessID.(string)
	if !ok {
		httperr.Internal(c)
		return
	}

	attempt, err := h.service.PayInvoice(
		c.Request.Context(),
		businessIDString,
		invoiceID,
		idempotencyKey,
		req.CardToken,
	)

	if err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, attempt)
}

func (h *PaymentHandler) GetPaymentAttempt(c *gin.Context) {
	attemptID := c.Param("id")

	businessID, exists := c.Get(middleware.BusinessIDKey)
	if !exists {
		httperr.Unauthorized(c, "business context missing")
		return
	}

	businessIDString, ok := businessID.(string)
	if !ok {
		httperr.Internal(c)
		return
	}

	attempt, err := h.service.GetPaymentAttempt(
		c.Request.Context(),
		businessIDString,
		attemptID,
	)
	if err != nil {
		writePaymentError(c, err)
		return
	}

	c.JSON(http.StatusOK, attempt)
}

func writePaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvoiceNotFound),
		errors.Is(err, service.ErrPaymentAttemptNotFound):
		httperr.NotFound(c, err.Error())

	case errors.Is(err, service.ErrInvoiceNotOwned),
		errors.Is(err, service.ErrPaymentAttemptNotOwned):
		httperr.Forbidden(c, err.Error())

	case errors.Is(err, service.ErrInvoiceAlreadyPaid),
		errors.Is(err, service.ErrInvoiceVoid),
		errors.Is(err, service.ErrInvoiceUncollectible),
		errors.Is(err, service.ErrPaymentInProgress),
		errors.Is(err, service.ErrIdempotencyConflict):
		httperr.Conflict(c, err.Error())

	default:
		httperr.Internal(c)
	}
}
