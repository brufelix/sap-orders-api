package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/service"
	"github.com/google/uuid"
)

type mockOrderRepo struct {
	orders map[uuid.UUID]domain.Order
}

func (m *mockOrderRepo) List(ctx context.Context, filter domain.OrderListFilter) (*domain.PagedResult[domain.Order], error) {
	result := make([]domain.Order, 0, len(m.orders))
	for _, order := range m.orders {
		if filter.Status != nil && order.Status != *filter.Status {
			continue
		}
		result = append(result, order)
	}
	return domain.NewPagedResult(result, filter.Page, filter.Limit, int64(len(result))), nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	order, ok := m.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &order, nil
}

func (m *mockOrderRepo) Create(ctx context.Context, orderNumber, createdBy string) (*domain.Order, error) {
	order := domain.Order{
		ID:          uuid.New(),
		OrderNumber: orderNumber,
		Status:      domain.OrderStatusOpen,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.orders[order.ID] = order
	return &order, nil
}

func (m *mockOrderRepo) Update(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	order, ok := m.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	order.Status = status
	m.orders[id] = order
	return &order, nil
}

type mockItemRepo struct {
	items map[uuid.UUID]domain.OrderItem
}

func (m *mockItemRepo) Create(ctx context.Context, orderID uuid.UUID, demandCode, description string, deliveryDate time.Time) (*domain.OrderItem, error) {
	item := domain.OrderItem{
		ID:           uuid.New(),
		OrderID:      orderID,
		DemandCode:   demandCode,
		Description:  description,
		DeliveryDate: deliveryDate,
		Status:       domain.ItemStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.items[item.ID] = item
	return &item, nil
}

func (m *mockItemRepo) GetByID(ctx context.Context, orderID, itemID uuid.UUID) (*domain.OrderItem, error) {
	item, ok := m.items[itemID]
	if !ok || item.OrderID != orderID {
		return nil, domain.ErrNotFound
	}
	return &item, nil
}

func (m *mockItemRepo) Update(ctx context.Context, orderID, itemID uuid.UUID, description *string, deliveryDate *time.Time, status *domain.ItemStatus) (*domain.OrderItem, error) {
	return m.GetByID(ctx, orderID, itemID)
}

func (m *mockItemRepo) UpdateStatus(ctx context.Context, itemID uuid.UUID, status domain.ItemStatus) error {
	item, ok := m.items[itemID]
	if !ok {
		return domain.ErrNotFound
	}
	item.Status = status
	m.items[itemID] = item
	return nil
}

func TestOrderService_CreateOrder_Validation(t *testing.T) {
	svc := service.NewOrderService(&mockOrderRepo{orders: map[uuid.UUID]domain.Order{}}, &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}})

	_, err := svc.CreateOrder(context.Background(), domain.CreateOrderInput{}, "user@example.com")
	if err != domain.ErrOrderNumberRequired {
		t.Fatalf("expected ErrOrderNumberRequired, got %v", err)
	}
}

func TestOrderService_AddItem_InvalidDeliveryDate(t *testing.T) {
	orderID := uuid.New()
	svc := service.NewOrderService(
		&mockOrderRepo{orders: map[uuid.UUID]domain.Order{
			orderID: {ID: orderID, OrderNumber: "PO-1"},
		}},
		&mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}},
	)

	_, err := svc.AddItem(context.Background(), orderID, domain.CreateOrderItemInput{
		DemandCode:   "DEM-1",
		Description:  "Test demand",
		DeliveryDate: "invalid-date",
	})
	if err != domain.ErrInvalidDeliveryDate {
		t.Fatalf("expected ErrInvalidDeliveryDate, got %v", err)
	}
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	svc := service.NewOrderService(&mockOrderRepo{orders: map[uuid.UUID]domain.Order{}}, &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}})

	order, err := svc.CreateOrder(context.Background(), domain.CreateOrderInput{OrderNumber: "PO-2026-001"}, "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.OrderNumber != "PO-2026-001" {
		t.Fatalf("expected order number PO-2026-001, got %s", order.OrderNumber)
	}
}
