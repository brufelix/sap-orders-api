package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusCompleted  OutboxStatus = "COMPLETED"
	OutboxStatusFailed     OutboxStatus = "FAILED"
	OutboxStatusCancelled  OutboxStatus = "CANCELLED"
)

type SyncStatusResponse struct {
	Outbox    OutboxEntry  `json:"outbox"`
	LatestLog *SAPSyncLog  `json:"latestLog,omitempty"`
}

type OutboxEntry struct {
	ID          uuid.UUID    `json:"id"`
	OrderItemID uuid.UUID    `json:"orderItemId"`
	OrderNumber string       `json:"orderNumber"`
	RFCFunction string       `json:"rfcFunction"`
	XMLPayload  string       `json:"xmlPayload"`
	Status      OutboxStatus `json:"status"`
	Attempts    int          `json:"attempts"`
	MaxAttempts int          `json:"maxAttempts"`
	ErrorMessage *string     `json:"errorMessage,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	ProcessedAt *time.Time   `json:"processedAt,omitempty"`
}
