package service

import (
	"context"
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

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

func (s *OrderService) ListOrders(ctx context.Context, filter domain.OrderListFilter) (*domain.PagedResult[domain.Order], error) {
	if filter.Page < 1 {
		filter.Page = defaultPage
	}
	if filter.Limit < 1 {
		filter.Limit = defaultLimit
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	return s.orders.List(ctx, filter)
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
	if !input.Status.IsValid() {
		return nil, domain.ErrInvalidOrderStatus
	}

	order, err := s.orders.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := order.Status.CanTransitionTo(*input.Status, order.Items); err != nil {
		return nil, err
	}

	return s.orders.Update(ctx, id, *input.Status)
}

func (s *OrderService) AddItem(ctx context.Context, orderID uuid.UUID, input domain.CreateOrderItemInput) (*domain.OrderItem, error) {
	if input.DemandCode == "" || input.Description == "" || input.DeliveryDate == "" {
		return nil, domain.ErrItemFieldsRequired
	}

	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == domain.OrderStatusClosed {
		return nil, domain.ErrOrderClosed
	}

	deliveryDate, err := time.Parse(time.DateOnly, input.DeliveryDate)
	if err != nil {
		return nil, domain.ErrInvalidDeliveryDate
	}
	if domain.IsDeliveryDateInPast(deliveryDate) {
		return nil, domain.ErrDeliveryDateInPast
	}

	item, err := s.items.Create(ctx, orderID, input.DemandCode, input.Description, deliveryDate)
	if err != nil {
		return nil, err
	}

	if order.Status == domain.OrderStatusOpen {
		if _, err := s.orders.Update(ctx, orderID, domain.OrderStatusInProgress); err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (s *OrderService) UpdateItem(ctx context.Context, orderID, itemID uuid.UUID, input domain.UpdateOrderItemInput) (*domain.OrderItem, error) {
	if input.Status != nil {
		return nil, domain.ErrItemStatusImmutable
	}
	if input.Description == nil && input.DeliveryDate == nil {
		return nil, domain.ErrValidation
	}

	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == domain.OrderStatusClosed {
		return nil, domain.ErrOrderClosed
	}

	item, err := s.items.GetByID(ctx, orderID, itemID)
	if err != nil {
		return nil, err
	}

	var deliveryDate *time.Time
	if input.DeliveryDate != nil {
		parsed, err := time.Parse(time.DateOnly, *input.DeliveryDate)
		if err != nil {
			return nil, domain.ErrInvalidDeliveryDate
		}
		if domain.IsDeliveryDateInPast(parsed) {
			return nil, domain.ErrDeliveryDateInPast
		}
		deliveryDate = &parsed
	}

	var status *domain.ItemStatus
	if item.Status.ShouldPromoteOnEdit() {
		updated := domain.ItemStatusUpdated
		status = &updated
	}

	return s.items.Update(ctx, orderID, itemID, input.Description, deliveryDate, status)
}
