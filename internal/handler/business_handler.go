package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/httperr"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

type BusinessHandler struct {
	service *service.BusinessService
}

func NewBusinessHandler(service *service.BusinessService) *BusinessHandler {
	return &BusinessHandler{
		service: service,
	}
}

type createBusinessRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *BusinessHandler) CreateBusiness(c *gin.Context) {
	var req createBusinessRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, "name is required")
		return
	}

	business, err := h.service.CreateBusiness(
		c.Request.Context(),
		req.Name,
	)
	if err != nil {
		httperr.Internal(c)
		return
	}

	c.JSON(http.StatusCreated, business)
}
