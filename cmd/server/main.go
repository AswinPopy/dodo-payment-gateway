package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/AswinPopy/dodo-payment-gateway/internal/database"
	"github.com/AswinPopy/dodo-payment-gateway/internal/handler"
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

	// Service
	businessService := service.NewBusinessService(businessRepo)

	// Handler
	businessHandler := handler.NewBusinessHandler(businessService)

	customerRepo := repository.NewCustomerRepository(db)
	customerService := service.NewCustomerService(
		customerRepo,
		businessRepo,
	)
	customerHandler := handler.NewCustomerHandler(customerService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.POST("/businesses", businessHandler.CreateBusiness)
	router.POST("/customers", customerHandler.CreateCustomer)

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
