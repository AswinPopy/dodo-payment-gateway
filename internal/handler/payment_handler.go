package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

func (h *PaymentHandler) PayInvoice(c *gin.Context) {
	invoiceID := c.Param("id")

	if invoiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invoice id is required",
		})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Idempotency-Key header is required",
		})
		return
	}

	businessID, exists := c.Get(middleware.BusinessIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "business context missing",
		})
		return
	}

	businessIDString, ok := businessID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid business context",
		})
		return
	}

	attempt, err := h.service.PayInvoice(
		c.Request.Context(),
		businessIDString,
		invoiceID,
		idempotencyKey,
	)

	if err != nil {
		switch err.Error() {
		case "invoice not found":
			c.JSON(http.StatusNotFound, gin.H{
				"error": "invoice not found",
			})

		case "invoice does not belong to business":
			c.JSON(http.StatusForbidden, gin.H{
				"error": "invoice does not belong to business",
			})

		case "invoice already paid":
			c.JSON(http.StatusConflict, gin.H{
				"error": "invoice already paid",
			})

		case "invoice is void":
			c.JSON(http.StatusConflict, gin.H{
				"error": "invoice is void",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to process payment",
			})
		}

		return
	}

	c.JSON(http.StatusOK, attempt)
}
