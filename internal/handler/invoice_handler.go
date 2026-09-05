package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
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
		httperr.BadRequest(c, "customer_id, currency and amount are required")
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

	invoice, err := h.service.CreateInvoice(
		c.Request.Context(),
		businessIDString,
		req.CustomerID,
		req.Currency,
		req.Amount,
	)

	if err != nil {
		switch err.Error() {
		case "business not found", "customer not found":
			httperr.NotFound(c, err.Error())
		case "customer does not belong to business":
			httperr.Forbidden(c, err.Error())
		case "amount must be greater than zero", "currency is required":
			httperr.BadRequest(c, err.Error())
		default:
			httperr.Internal(c)
		}
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	invoiceID := c.Param("id")

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

	invoice, err := h.service.GetInvoice(
		c.Request.Context(),
		businessIDString,
		invoiceID,
	)
	if err != nil {
		switch err.Error() {
		case "invoice not found":
			httperr.NotFound(c, err.Error())
		case "invoice does not belong to business":
			httperr.Forbidden(c, err.Error())
		default:
			httperr.Internal(c)
		}
		return
	}

	c.JSON(http.StatusOK, invoice)
}
