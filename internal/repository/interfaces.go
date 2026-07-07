package repository

import (
	"context"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
)

type OrderRepository interface {
	List(ctx context.Context, filter domain.OrderListFilter) (*domain.PagedResult[domain.Order], error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	Create(ctx context.Context, orderNumber, createdBy string) (*domain.Order, error)
	Update(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error)
}

type ItemRepository interface {
	Create(ctx context.Context, orderID uuid.UUID, demandCode, description string, deliveryDate time.Time) (*domain.OrderItem, error)
	GetByID(ctx context.Context, orderID, itemID uuid.UUID) (*domain.OrderItem, error)
	Update(ctx context.Context, orderID, itemID uuid.UUID, description *string, deliveryDate *time.Time, status *domain.ItemStatus) (*domain.OrderItem, error)
	UpdateStatus(ctx context.Context, itemID uuid.UUID, status domain.ItemStatus) error
}

type SyncRepository interface {
	Create(ctx context.Context, log domain.SAPSyncLog) (*domain.SAPSyncLog, error)
	ListByItemID(ctx context.Context, itemID uuid.UUID) ([]domain.SAPSyncLog, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, entry domain.OutboxEntry) (*domain.OutboxEntry, error)
	GetByID(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error)
	GetLatestByItemID(ctx context.Context, itemID uuid.UUID) (*domain.OutboxEntry, error)
	ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEntry, error)
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string, retry bool) error
	Cancel(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error)
}

type Transactor interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
