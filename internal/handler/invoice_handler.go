package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type InvoiceHandler struct {
	service *service.InvoiceService
}

func NewInvoiceHandler(service *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

type createInvoiceRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	Currency   string `json:"currency" binding:"required"`
	Amount     int64  `json:"amount" binding:"required"`
}

func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	var req createInvoiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "customer_id, currency and amount are required",
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

	invoice, err := h.service.CreateInvoice(
		c.Request.Context(),
		businessIDString,
		req.CustomerID,
		req.Currency,
		req.Amount,
	)

	if err != nil {
		switch err.Error() {
		case "business not found":
			c.JSON(http.StatusNotFound, gin.H{
				"error": "business not found",
			})
		case "customer not found":
			c.JSON(http.StatusNotFound, gin.H{
				"error": "customer not found",
			})
		case "customer does not belong to business":
			c.JSON(http.StatusForbidden, gin.H{
				"error": "customer does not belong to business",
			})
		case "amount must be greater than zero":
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "amount must be greater than zero",
			})
		case "currency is required":
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "currency is required",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create invoice",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, invoice)
}
