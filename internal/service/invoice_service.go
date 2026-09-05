package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
)

type InvoiceService struct {
	invoiceRepo  *repository.InvoiceRepository
	businessRepo *repository.BusinessRepository
	customerRepo *repository.CustomerRepository
}

func NewInvoiceService(
	invoiceRepo *repository.InvoiceRepository,
	businessRepo *repository.BusinessRepository,
	customerRepo *repository.CustomerRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  invoiceRepo,
		businessRepo: businessRepo,
		customerRepo: customerRepo,
	}
}

func (s *InvoiceService) CreateInvoice(
	ctx context.Context,
	businessID string,
	customerID string,
	currency string,
	amount int64,
) (*model.Invoice, error) {

	// 1. Validate amount.
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	// 2. Validate currency.
	currency = strings.ToUpper(strings.TrimSpace(currency))

	if currency == "" {
		return nil, errors.New("currency is required")
	}

	// 3. Check that the business exists.
	businessExists, err := s.businessRepo.Exists(ctx, businessID)
	if err != nil {
		return nil, err
	}

	if !businessExists {
		return nil, errors.New("business not found")
	}

	// 4. Get the customer.
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, errors.New("customer not found")
	}

	// 5. Make sure the customer belongs to this business.
	if customer.BusinessID != businessID {
		return nil, errors.New("customer does not belong to business")
	}

	// 6. Create the invoice.
	invoice := &model.Invoice{
		ID:         uuid.New().String(),
		BusinessID: businessID,
		CustomerID: customerID,
		Currency:   currency,
		Amount:     amount,
		Status:     model.InvoiceStatusDraft,
	}

	// 7. Persist it.
	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}
