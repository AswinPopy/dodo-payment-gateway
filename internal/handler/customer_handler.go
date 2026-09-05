package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "business_id, name and valid email are required",
		})
		return
	}
	businessID, exists := c.Get(middleware.BusinessIDKey)
	businessIDString, ok := businessID.(string)

	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid business context",
		})
		return
	}
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "business context missing",
		})
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
			c.JSON(http.StatusNotFound, gin.H{
				"error": "business not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create customer",
		})
		return
	}

	c.JSON(http.StatusCreated, customer)
}
