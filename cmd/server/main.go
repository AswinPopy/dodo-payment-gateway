package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/database"
	"github.com/AswinPopy/dodo-payment-gateway/internal/handler"
	"github.com/AswinPopy/dodo-payment-gateway/internal/middleware"
	"github.com/AswinPopy/dodo-payment-gateway/internal/psp"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
	"github.com/AswinPopy/dodo-payment-gateway/internal/service"
	"github.com/AswinPopy/dodo-payment-gateway/internal/webhook"
	"github.com/AswinPopy/dodo-payment-gateway/internal/worker"
)

func main() {
	ctx := context.Background()

	db, err := database.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}
	if err := database.RunMigrations(ctx, db, migrationsPath); err != nil {
		log.Fatal(err)
	}

	// Repository
	businessRepo := repository.NewBusinessRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	paymentAttemptRepo := repository.NewPaymentAttemptRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	webhookEventRepo := repository.NewWebhookEventRepository(db)
	outboundEventRepo := repository.NewOutboundEventRepository(db)

	pspURL := os.Getenv("PSP_BASE_URL")
	if pspURL == "" {
		pspURL = "http://localhost:8081"
	}
	pspClient := psp.NewHTTPClient(pspURL)

	pspWebhookSecret := os.Getenv("PSP_WEBHOOK_SECRET")
	if pspWebhookSecret == "" {
		pspWebhookSecret = webhook.DefaultPSPSecret
	}

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

	paymentService := service.NewPaymentService(
		db,
		invoiceRepo,
		paymentAttemptRepo,
		idempotencyRepo,
		webhookEventRepo,
		outboundEventRepo,
		pspWebhookSecret,
		pspClient,
	)
	webhookService := service.NewWebhookService(
		businessRepo,
		outboundEventRepo,
	)

	// Handler
	businessHandler := handler.NewBusinessHandler(businessService)
	customerHandler := handler.NewCustomerHandler(customerService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	webhookHandler := handler.NewWebhookHandler(paymentService, webhookService)

	go worker.NewWebhookDispatcher(
		businessRepo,
		outboundEventRepo,
	).Run(ctx)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())

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
	router.POST(
		"/api-keys",
		middleware.OptionalAPIKeyAuth(apiKeyRepo),
		apiKeyHandler.CreateAPIKey,
	)
	router.GET(
		"/api-keys",
		middleware.APIKeyAuth(apiKeyRepo),
		apiKeyHandler.ListAPIKeys,
	)
	router.POST(
		"/api-keys/:id/revoke",
		middleware.APIKeyAuth(apiKeyRepo),
		apiKeyHandler.RevokeAPIKey,
	)
	router.POST(
		"/api-keys/:id/rotate",
		middleware.APIKeyAuth(apiKeyRepo),
		apiKeyHandler.RotateAPIKey,
	)
	router.POST(
		"/invoices",
		middleware.APIKeyAuth(apiKeyRepo),
		invoiceHandler.CreateInvoice,
	)
	router.GET(
		"/invoices/:id",
		middleware.APIKeyAuth(apiKeyRepo),
		invoiceHandler.GetInvoice,
	)

	router.POST(
		"/invoices/:id/pay",
		middleware.APIKeyAuth(apiKeyRepo),
		paymentHandler.PayInvoice,
	)
	router.GET(
		"/payment-attempts/:id",
		middleware.APIKeyAuth(apiKeyRepo),
		paymentHandler.GetPaymentAttempt,
	)

	router.POST(
		"/webhook-endpoint",
		middleware.APIKeyAuth(apiKeyRepo),
		webhookHandler.SetEndpoint,
	)
	router.GET(
		"/events",
		middleware.APIKeyAuth(apiKeyRepo),
		webhookHandler.ListEvents,
	)
	router.GET(
		"/events/:id",
		middleware.APIKeyAuth(apiKeyRepo),
		webhookHandler.GetEvent,
	)
	router.POST(
		"/events/:id/redeliver",
		middleware.APIKeyAuth(apiKeyRepo),
		webhookHandler.RedeliverEvent,
	)

	router.POST("/webhooks/psp", webhookHandler.HandlePSPWebhook)

	slog.Info("invoice api listening", "addr", ":8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
