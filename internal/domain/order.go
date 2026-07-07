package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusOpen       OrderStatus = "OPEN"
	OrderStatusInProgress OrderStatus = "IN_PROGRESS"
	OrderStatusClosed     OrderStatus = "CLOSED"
)

type ItemStatus string

const (
	ItemStatusPending   ItemStatus = "PENDING"
	ItemStatusUpdated   ItemStatus = "UPDATED"
	ItemStatusSynced    ItemStatus = "SYNCED"
	ItemStatusFailed    ItemStatus = "FAILED"
)

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "PENDING"
	SyncStatusSuccess SyncStatus = "SUCCESS"
	SyncStatusFailed  SyncStatus = "FAILED"
)

type Order struct {
	ID          uuid.UUID   `json:"id"`
	OrderNumber string      `json:"orderNumber"`
	Status      OrderStatus `json:"status"`
	CreatedBy   string      `json:"createdBy"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Items       []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID           uuid.UUID  `json:"id"`
	OrderID      uuid.UUID  `json:"orderId"`
	DemandCode   string     `json:"demandCode"`
	Description  string     `json:"description"`
	DeliveryDate time.Time  `json:"deliveryDate"`
	Status       ItemStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type SAPSyncLog struct {
	ID          uuid.UUID  `json:"id"`
	OrderItemID uuid.UUID  `json:"orderItemId"`
	RFCFunction string     `json:"rfcFunction"`
	XMLRequest  string     `json:"xmlRequest"`
	XMLResponse *string    `json:"xmlResponse,omitempty"`
	Status      SyncStatus `json:"status"`
	ErrorMessage *string   `json:"errorMessage,omitempty"`
	SyncedAt    time.Time  `json:"syncedAt"`
}

type CreateOrderInput struct {
	OrderNumber string `json:"orderNumber"`
}

type UpdateOrderInput struct {
	Status *OrderStatus `json:"status"`
}

type CreateOrderItemInput struct {
	DemandCode   string `json:"demandCode"`
	Description  string `json:"description"`
	DeliveryDate string `json:"deliveryDate"`
}

type UpdateOrderItemInput struct {
	Description  *string     `json:"description"`
	DeliveryDate *string     `json:"deliveryDate"`
	Status       *ItemStatus `json:"status"`
}
