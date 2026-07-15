package domain_test

import (
	"testing"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
)

func TestOrderStatus_CanTransitionTo(t *testing.T) {
	item := domain.OrderItem{Status: domain.ItemStatusSynced}

	tests := []struct {
		name    string
		from    domain.OrderStatus
		to      domain.OrderStatus
		items   []domain.OrderItem
		wantErr error
	}{
		{
			name:  "open to in progress with items",
			from:  domain.OrderStatusOpen,
			to:    domain.OrderStatusInProgress,
			items: []domain.OrderItem{{ID: uuid.New()}},
		},
		{
			name:    "open to in progress without items",
			from:    domain.OrderStatusOpen,
			to:      domain.OrderStatusInProgress,
			wantErr: domain.ErrInvalidStatusTransition,
		},
		{
			name:  "in progress to closed when all terminal",
			from:  domain.OrderStatusInProgress,
			to:    domain.OrderStatusClosed,
			items: []domain.OrderItem{item, {Status: domain.ItemStatusFailed}},
		},
		{
			name:    "in progress to closed with pending item",
			from:    domain.OrderStatusInProgress,
			to:      domain.OrderStatusClosed,
			items:   []domain.OrderItem{{Status: domain.ItemStatusPending}},
			wantErr: domain.ErrInvalidStatusTransition,
		},
		{
			name:    "closed cannot change",
			from:    domain.OrderStatusClosed,
			to:      domain.OrderStatusOpen,
			wantErr: domain.ErrOrderClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.from.CanTransitionTo(tt.to, tt.items)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestItemStatus_CanSync(t *testing.T) {
	if !domain.ItemStatusUpdated.CanSync() {
		t.Fatal("UPDATED should be syncable")
	}
	if !domain.ItemStatusFailed.CanSync() {
		t.Fatal("FAILED should be syncable")
	}
	if domain.ItemStatusPending.CanSync() {
		t.Fatal("PENDING should not be syncable")
	}
}

func TestIsDeliveryDateInPast(t *testing.T) {
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)

	if !domain.IsDeliveryDateInPast(yesterday) {
		t.Fatal("yesterday should be in the past")
	}
	if domain.IsDeliveryDateInPast(tomorrow) {
		t.Fatal("tomorrow should not be in the past")
	}
}
