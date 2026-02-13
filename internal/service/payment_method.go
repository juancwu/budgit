package service

import (
	"fmt"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"github.com/google/uuid"
)

type CreatePaymentMethodDTO struct {
	SpaceID   string
	Name      string
	Type      model.PaymentMethodType
	LastFour  string
	CreatedBy string
}

type UpdatePaymentMethodDTO struct {
	ID       string
	Name     string
	Type     model.PaymentMethodType
	LastFour string
}

type PaymentMethodService struct {
	methodRepo repository.PaymentMethodRepository
}

func NewPaymentMethodService(methodRepo repository.PaymentMethodRepository) *PaymentMethodService {
	return &PaymentMethodService{
		methodRepo: methodRepo,
	}
}

func (s *PaymentMethodService) CreateMethod(dto CreatePaymentMethodDTO) (*model.PaymentMethod, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, fmt.Errorf("payment method name cannot be empty")
	}
	if dto.Type != model.PaymentMethodTypeCredit && dto.Type != model.PaymentMethodTypeDebit {
		return nil, fmt.Errorf("invalid payment method type")
	}
	if len(dto.LastFour) != 4 {
		return nil, fmt.Errorf("last four digits must be exactly 4 characters")
	}

	now := time.Now()
	method := &model.PaymentMethod{
		ID:        uuid.NewString(),
		SpaceID:   dto.SpaceID,
		Name:      name,
		Type:      dto.Type,
		LastFour:  &dto.LastFour,
		CreatedBy: dto.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.methodRepo.Create(method)
	if err != nil {
		return nil, err
	}

	return method, nil
}

func (s *PaymentMethodService) GetMethodsForSpace(spaceID string) ([]*model.PaymentMethod, error) {
	return s.methodRepo.GetBySpaceID(spaceID)
}

func (s *PaymentMethodService) GetMethod(id string) (*model.PaymentMethod, error) {
	return s.methodRepo.GetByID(id)
}

func (s *PaymentMethodService) UpdateMethod(dto UpdatePaymentMethodDTO) (*model.PaymentMethod, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, fmt.Errorf("payment method name cannot be empty")
	}
	if dto.Type != model.PaymentMethodTypeCredit && dto.Type != model.PaymentMethodTypeDebit {
		return nil, fmt.Errorf("invalid payment method type")
	}
	if len(dto.LastFour) != 4 {
		return nil, fmt.Errorf("last four digits must be exactly 4 characters")
	}

	method, err := s.methodRepo.GetByID(dto.ID)
	if err != nil {
		return nil, err
	}

	method.Name = name
	method.Type = dto.Type
	method.LastFour = &dto.LastFour

	err = s.methodRepo.Update(method)
	if err != nil {
		return nil, err
	}

	return method, nil
}

func (s *PaymentMethodService) DeleteMethod(id string) error {
	return s.methodRepo.Delete(id)
}
