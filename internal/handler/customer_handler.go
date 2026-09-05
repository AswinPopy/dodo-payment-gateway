package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type CustomerHandler struct {
	service *service.CustomerService
}

func NewCustomerHandler(service *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		service: service,
	}
}

type createCustomerRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req createCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, "name and a valid email are required")
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

	customer, err := h.service.CreateCustomer(
		c.Request.Context(),
		businessIDString,
		req.Name,
		req.Email,
	)

	if err != nil {
		if err.Error() == "business not found" {
			httperr.NotFound(c, "business not found")
			return
		}

		httperr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, customer)
}
