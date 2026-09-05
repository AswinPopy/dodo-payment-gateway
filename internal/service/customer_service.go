package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/AswinPopy/dodo-payment-gateway/internal/model"
	"github.com/AswinPopy/dodo-payment-gateway/internal/repository"
)

type CustomerService struct {
	customerRepo *repository.CustomerRepository
	businessRepo *repository.BusinessRepository
}

func NewCustomerService(
	customerRepo *repository.CustomerRepository,
	businessRepo *repository.BusinessRepository,
) *CustomerService {
	return &CustomerService{
		customerRepo: customerRepo,
		businessRepo: businessRepo,
	}
}

func (s *CustomerService) CreateCustomer(
	ctx context.Context,
	businessID string,
	name string,
	email string,
) (*model.Customer, error) {

	exists, err := s.businessRepo.Exists(ctx, businessID)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("business not found")
	}

	customer := &model.Customer{
		ID:         uuid.New().String(),
		BusinessID: businessID,
		Name:       name,
		Email:      email,
	}

	if err := s.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}
