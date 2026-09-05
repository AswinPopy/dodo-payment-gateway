package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/database"
	"github.com/AswinPopy/dodo-payment-gateway/internal/handler"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
)

func main() {
	ctx := context.Background()

	db, err := database.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Repository
	businessRepo := repository.NewBusinessRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)

	// Service
	businessService := service.NewBusinessService(businessRepo)
	customerService := service.NewCustomerService(customerRepo, businessRepo)
	apiKeyService := service.NewAPIKeyService(
		apiKeyRepo,
		businessRepo,
	)
	invoiceService := service.NewInvoiceService(
		invoiceRepo,
		businessRepo,
		customerRepo,
	)

	// Handler
	businessHandler := handler.NewBusinessHandler(businessService)
	customerHandler := handler.NewCustomerHandler(customerService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.POST("/businesses", businessHandler.CreateBusiness)
	router.POST(
		"/customers",
		middleware.APIKeyAuth(apiKeyRepo),
		customerHandler.CreateCustomer,
	)
	router.POST("/api-keys", apiKeyHandler.CreateAPIKey)
	router.POST(
		"/invoices",
		middleware.APIKeyAuth(apiKeyRepo),
		invoiceHandler.CreateInvoice,
	)
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

//ba649d53-a762-4920-9bc9-60338979e3b6 | aa1e5c14-0fcc-46be-85f6-7c5262c03087 | 9ae51207d033826f9354a2a7a40e503d018ef182bb3d788eadd3d775d3e4d0db | 2026-09-05 09:14:45.340677+00 |

//curl -X POST http://localhost:8080/customers \
//   -H "Content-Type: application/json" \
//   -H "Authorization: Bearer 34e9556d6428ee04d62bbbe38fbbb9366485dab5c1f5f2fbcd189dfb23a73c2d" \
//   -d '{
//     "business_id": " aa1e5c14-0fcc-46be-85f6-7c5262c03087",
//     "name": "Jane Doe",
//     "email": "jane@example.com"
//   }'
//dodo_live_f82f4df1770fbcf43f3215423388bbb99a53dfe22851809aa5235a56c89f2487
