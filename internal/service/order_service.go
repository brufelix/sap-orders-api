package service

import (
	"context"
	"fmt"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/repository"
	"github.com/google/uuid"
)

type OrderService struct {
	orders repository.OrderRepository
	items  repository.ItemRepository
}

func NewOrderService(orders repository.OrderRepository, items repository.ItemRepository) *OrderService {
	return &OrderService{orders: orders, items: items}
}

func (s *OrderService) ListOrders(ctx context.Context) ([]domain.Order, error) {
	return s.orders.List(ctx)
}

func (s *OrderService) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return s.orders.GetByID(ctx, id)
}

func (s *OrderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput, createdBy string) (*domain.Order, error) {
	if input.OrderNumber == "" {
		return nil, domain.ErrOrderNumberRequired
	}
	return s.orders.Create(ctx, input.OrderNumber, createdBy)
}

func (s *OrderService) UpdateOrder(ctx context.Context, id uuid.UUID, input domain.UpdateOrderInput) (*domain.Order, error) {
	if input.Status == nil {
		return nil, domain.ErrStatusRequired
	}
	return s.orders.Update(ctx, id, *input.Status)
}

func (s *OrderService) AddItem(ctx context.Context, orderID uuid.UUID, input domain.CreateOrderItemInput) (*domain.OrderItem, error) {
	if input.DemandCode == "" || input.Description == "" || input.DeliveryDate == "" {
		return nil, domain.ErrItemFieldsRequired
	}

	if _, err := s.orders.GetByID(ctx, orderID); err != nil {
		return nil, err
	}

	deliveryDate, err := time.Parse(time.DateOnly, input.DeliveryDate)
	if err != nil {
		return nil, domain.ErrInvalidDeliveryDate
	}

	return s.items.Create(ctx, orderID, input.DemandCode, input.Description, deliveryDate)
}

func (s *OrderService) UpdateItem(ctx context.Context, orderID, itemID uuid.UUID, input domain.UpdateOrderItemInput) (*domain.OrderItem, error) {
	var deliveryDate *time.Time
	if input.DeliveryDate != nil {
		parsed, err := time.Parse(time.DateOnly, *input.DeliveryDate)
		if err != nil {
			return nil, domain.ErrInvalidDeliveryDate
		}
		deliveryDate = &parsed
	}

	return s.items.Update(ctx, orderID, itemID, input.Description, deliveryDate, input.Status)
}
