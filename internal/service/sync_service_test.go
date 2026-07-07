package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/repository"
	"github.com/brufelix/sap-orders-api/internal/sap"
	"github.com/brufelix/sap-orders-api/internal/service"
	"github.com/google/uuid"
	"log/slog"
	"os"
)

type mockOutboxRepo struct {
	entries map[uuid.UUID]domain.OutboxEntry
}

func (m *mockOutboxRepo) Create(ctx context.Context, entry domain.OutboxEntry) (*domain.OutboxEntry, error) {
	entry.ID = uuid.New()
	m.entries[entry.ID] = entry
	return &entry, nil
}

func (m *mockOutboxRepo) GetByID(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error) {
	entry, ok := m.entries[outboxID]
	if !ok || entry.OrderItemID != itemID {
		return nil, domain.ErrNotFound
	}
	return &entry, nil
}

func (m *mockOutboxRepo) GetLatestByItemID(ctx context.Context, itemID uuid.UUID) (*domain.OutboxEntry, error) {
	var latest *domain.OutboxEntry
	for _, entry := range m.entries {
		if entry.OrderItemID != itemID {
			continue
		}
		if latest == nil || entry.CreatedAt.After(latest.CreatedAt) {
			copy := entry
			latest = &copy
		}
	}
	if latest == nil {
		return nil, domain.ErrNotFound
	}
	return latest, nil
}

func (m *mockOutboxRepo) ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEntry, error) {
	return nil, nil
}

func (m *mockOutboxRepo) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string, retry bool) error {
	return nil
}

func (m *mockOutboxRepo) Cancel(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error) {
	entry, err := m.GetByID(ctx, itemID, outboxID)
	if err != nil {
		return nil, err
	}
	if entry.Status != domain.OutboxStatusPending {
		return nil, domain.ErrSyncNotCancellable
	}
	entry.Status = domain.OutboxStatusCancelled
	m.entries[outboxID] = *entry
	return entry, nil
}

type mockSyncRepo struct{}

func (m *mockSyncRepo) Create(ctx context.Context, log domain.SAPSyncLog) (*domain.SAPSyncLog, error) {
	return &log, nil
}

func (m *mockSyncRepo) ListByItemID(ctx context.Context, itemID uuid.UUID) ([]domain.SAPSyncLog, error) {
	return nil, nil
}

type mockTransactor struct{}

func (m *mockTransactor) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newSyncService(orderID, itemID uuid.UUID, outbox *mockOutboxRepo) *service.SyncService {
	return service.NewSyncService(
		&mockOrderRepo{orders: map[uuid.UUID]domain.Order{}},
		&mockItemRepo{items: map[uuid.UUID]domain.OrderItem{
			itemID: {ID: itemID, OrderID: orderID, Status: domain.ItemStatusUpdated},
		}},
		&mockSyncRepo{},
		outbox,
		&mockTransactor{},
		sap.NewStubClient(slog.New(slog.NewTextHandler(os.Stdout, nil))),
		"Z_UPDATE_DEMAND",
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
	)
}

func TestSyncService_GetLatestSyncStatus(t *testing.T) {
	orderID := uuid.New()
	itemID := uuid.New()
	outboxID := uuid.New()
	outbox := &mockOutboxRepo{entries: map[uuid.UUID]domain.OutboxEntry{
		outboxID: {
			ID:          outboxID,
			OrderItemID: itemID,
			Status:      domain.OutboxStatusPending,
			CreatedAt:   time.Now(),
		},
	}}

	svc := newSyncService(orderID, itemID, outbox)
	status, err := svc.GetLatestSyncStatus(context.Background(), orderID, itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Outbox.ID != outboxID {
		t.Fatalf("expected outbox %s, got %s", outboxID, status.Outbox.ID)
	}
}

func TestSyncService_CancelSync_NotCancellable(t *testing.T) {
	orderID := uuid.New()
	itemID := uuid.New()
	outboxID := uuid.New()
	outbox := &mockOutboxRepo{entries: map[uuid.UUID]domain.OutboxEntry{
		outboxID: {
			ID:          outboxID,
			OrderItemID: itemID,
			Status:      domain.OutboxStatusProcessing,
			CreatedAt:   time.Now(),
		},
	}}

	svc := newSyncService(orderID, itemID, outbox)
	_, err := svc.CancelSync(context.Background(), orderID, itemID, outboxID)
	if err != domain.ErrSyncNotCancellable {
		t.Fatalf("expected ErrSyncNotCancellable, got %v", err)
	}
}

func TestSyncService_CancelSync_Success(t *testing.T) {
	orderID := uuid.New()
	itemID := uuid.New()
	outboxID := uuid.New()
	outbox := &mockOutboxRepo{entries: map[uuid.UUID]domain.OutboxEntry{
		outboxID: {
			ID:          outboxID,
			OrderItemID: itemID,
			Status:      domain.OutboxStatusPending,
			CreatedAt:   time.Now(),
		},
	}}

	svc := newSyncService(orderID, itemID, outbox)
	entry, err := svc.CancelSync(context.Background(), orderID, itemID, outboxID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Status != domain.OutboxStatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", entry.Status)
	}
}

var _ repository.OutboxRepository = (*mockOutboxRepo)(nil)
