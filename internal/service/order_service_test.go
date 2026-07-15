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
	items  *mockItemRepo
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
	if m.items != nil {
		order.Items = nil
		for _, item := range m.items.items {
			if item.OrderID == id {
				order.Items = append(order.Items, item)
			}
		}
	}
	copy := order
	return &copy, nil
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
	copy := item
	return &copy, nil
}

func (m *mockItemRepo) Update(ctx context.Context, orderID, itemID uuid.UUID, description *string, deliveryDate *time.Time, status *domain.ItemStatus) (*domain.OrderItem, error) {
	item, err := m.GetByID(ctx, orderID, itemID)
	if err != nil {
		return nil, err
	}
	if description != nil {
		item.Description = *description
	}
	if deliveryDate != nil {
		item.DeliveryDate = *deliveryDate
	}
	if status != nil {
		item.Status = *status
	}
	m.items[itemID] = *item
	return item, nil
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

func futureDate() string {
	return time.Now().UTC().AddDate(0, 0, 7).Format(time.DateOnly)
}

func TestOrderService_CreateOrder_Validation(t *testing.T) {
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	svc := service.NewOrderService(&mockOrderRepo{orders: map[uuid.UUID]domain.Order{}, items: items}, items)

	_, err := svc.CreateOrder(context.Background(), domain.CreateOrderInput{}, "user@example.com")
	if err != domain.ErrOrderNumberRequired {
		t.Fatalf("expected ErrOrderNumberRequired, got %v", err)
	}
}

func TestOrderService_AddItem_InvalidDeliveryDate(t *testing.T) {
	orderID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	svc := service.NewOrderService(
		&mockOrderRepo{orders: map[uuid.UUID]domain.Order{
			orderID: {ID: orderID, OrderNumber: "PO-1", Status: domain.OrderStatusOpen},
		}, items: items},
		items,
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
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	svc := service.NewOrderService(&mockOrderRepo{orders: map[uuid.UUID]domain.Order{}, items: items}, items)

	order, err := svc.CreateOrder(context.Background(), domain.CreateOrderInput{OrderNumber: "PO-2026-001"}, "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.OrderNumber != "PO-2026-001" {
		t.Fatalf("expected order number PO-2026-001, got %s", order.OrderNumber)
	}
}

func TestOrderService_AddItem_ClosedOrder(t *testing.T) {
	orderID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	orders := &mockOrderRepo{orders: map[uuid.UUID]domain.Order{
		orderID: {ID: orderID, OrderNumber: "PO-1", Status: domain.OrderStatusClosed},
	}, items: items}
	svc := service.NewOrderService(orders, items)

	_, err := svc.AddItem(context.Background(), orderID, domain.CreateOrderItemInput{
		DemandCode:   "DEM-1",
		Description:  "Test",
		DeliveryDate: futureDate(),
	})
	if err != domain.ErrOrderClosed {
		t.Fatalf("expected ErrOrderClosed, got %v", err)
	}
}

func TestOrderService_AddItem_PromotesOrderToInProgress(t *testing.T) {
	orderID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	orders := &mockOrderRepo{orders: map[uuid.UUID]domain.Order{
		orderID: {ID: orderID, OrderNumber: "PO-1", Status: domain.OrderStatusOpen},
	}, items: items}
	svc := service.NewOrderService(orders, items)

	_, err := svc.AddItem(context.Background(), orderID, domain.CreateOrderItemInput{
		DemandCode:   "DEM-1",
		Description:  "Test",
		DeliveryDate: futureDate(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders.orders[orderID].Status != domain.OrderStatusInProgress {
		t.Fatalf("expected IN_PROGRESS, got %s", orders.orders[orderID].Status)
	}
}

func TestOrderService_UpdateOrder_InvalidTransition(t *testing.T) {
	orderID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{}}
	orders := &mockOrderRepo{orders: map[uuid.UUID]domain.Order{
		orderID: {ID: orderID, OrderNumber: "PO-1", Status: domain.OrderStatusOpen},
	}, items: items}
	svc := service.NewOrderService(orders, items)

	status := domain.OrderStatusClosed
	_, err := svc.UpdateOrder(context.Background(), orderID, domain.UpdateOrderInput{Status: &status})
	if err != domain.ErrInvalidStatusTransition {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestOrderService_UpdateItem_RejectsStatusField(t *testing.T) {
	orderID := uuid.New()
	itemID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{
		itemID: {
			ID: itemID, OrderID: orderID, Status: domain.ItemStatusPending,
			DeliveryDate: time.Now().UTC().AddDate(0, 0, 7),
		},
	}}
	orders := &mockOrderRepo{orders: map[uuid.UUID]domain.Order{
		orderID: {ID: orderID, Status: domain.OrderStatusInProgress},
	}, items: items}
	svc := service.NewOrderService(orders, items)

	status := domain.ItemStatusSynced
	_, err := svc.UpdateItem(context.Background(), orderID, itemID, domain.UpdateOrderItemInput{Status: &status})
	if err != domain.ErrItemStatusImmutable {
		t.Fatalf("expected ErrItemStatusImmutable, got %v", err)
	}
}

func TestOrderService_UpdateItem_PromotesToUpdated(t *testing.T) {
	orderID := uuid.New()
	itemID := uuid.New()
	items := &mockItemRepo{items: map[uuid.UUID]domain.OrderItem{
		itemID: {
			ID: itemID, OrderID: orderID, Status: domain.ItemStatusSynced,
			Description:  "old",
			DeliveryDate: time.Now().UTC().AddDate(0, 0, 7),
		},
	}}
	orders := &mockOrderRepo{orders: map[uuid.UUID]domain.Order{
		orderID: {ID: orderID, Status: domain.OrderStatusInProgress},
	}, items: items}
	svc := service.NewOrderService(orders, items)

	description := "new description"
	item, err := svc.UpdateItem(context.Background(), orderID, itemID, domain.UpdateOrderItemInput{Description: &description})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Status != domain.ItemStatusUpdated {
		t.Fatalf("expected UPDATED, got %s", item.Status)
	}
}
