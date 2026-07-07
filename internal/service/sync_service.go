package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/repository"
	"github.com/brufelix/sap-orders-api/internal/sap"
	"github.com/google/uuid"
)

type SyncService struct {
	orders      repository.OrderRepository
	items       repository.ItemRepository
	syncLogs    repository.SyncRepository
	outbox      repository.OutboxRepository
	transactor  repository.Transactor
	sapClient   sap.Client
	rfcFunction string
	logger      *slog.Logger
}

func NewSyncService(
	orders repository.OrderRepository,
	items repository.ItemRepository,
	syncLogs repository.SyncRepository,
	outbox repository.OutboxRepository,
	transactor repository.Transactor,
	sapClient sap.Client,
	rfcFunction string,
	logger *slog.Logger,
) *SyncService {
	return &SyncService{
		orders:      orders,
		items:       items,
		syncLogs:    syncLogs,
		outbox:      outbox,
		transactor:  transactor,
		sapClient:   sapClient,
		rfcFunction: rfcFunction,
		logger:      logger,
	}
}

func (s *SyncService) EnqueueSync(ctx context.Context, orderID, itemID uuid.UUID) (*domain.OutboxEntry, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	item, err := s.items.GetByID(ctx, orderID, itemID)
	if err != nil {
		return nil, err
	}

	xmlPayload, err := sap.BuildDemandUpdateXML(order.OrderNumber, *item)
	if err != nil {
		return nil, err
	}

	var created *domain.OutboxEntry
	err = s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.items.UpdateStatus(txCtx, itemID, domain.ItemStatusUpdated); err != nil {
			return err
		}

		entry, err := s.outbox.Create(txCtx, domain.OutboxEntry{
			OrderItemID: itemID,
			OrderNumber: order.OrderNumber,
			RFCFunction: s.rfcFunction,
			XMLPayload:  xmlPayload,
			Status:      domain.OutboxStatusPending,
			MaxAttempts: 3,
		})
		if err != nil {
			return err
		}

		created = entry
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.Info("sync enqueued",
		"order_id", orderID,
		"item_id", itemID,
		"outbox_id", created.ID,
	)

	return created, nil
}

func (s *SyncService) ProcessOutboxEntry(ctx context.Context, entry domain.OutboxEntry) error {
	result, err := s.sapClient.SyncDemandUpdate(ctx, entry.RFCFunction, entry.XMLPayload)
	if err != nil {
		return s.handleSyncFailure(ctx, entry, err.Error(), entry.Attempts < entry.MaxAttempts)
	}

	status := domain.SyncStatusSuccess
	itemStatus := domain.ItemStatusSynced
	errorMessage := ""
	if !result.Success {
		status = domain.SyncStatusFailed
		itemStatus = domain.ItemStatusFailed
		errorMessage = result.Message
	}

	return s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		response := result.XMLResponse
		logEntry := domain.SAPSyncLog{
			OrderItemID: entry.OrderItemID,
			RFCFunction: entry.RFCFunction,
			XMLRequest:  entry.XMLPayload,
			XMLResponse: &response,
			Status:      status,
		}
		if errorMessage != "" {
			logEntry.ErrorMessage = &errorMessage
		}

		if _, err := s.syncLogs.Create(txCtx, logEntry); err != nil {
			return err
		}

		if err := s.items.UpdateStatus(txCtx, entry.OrderItemID, itemStatus); err != nil {
			return err
		}

		if status == domain.SyncStatusSuccess {
			return s.outbox.MarkCompleted(txCtx, entry.ID)
		}

		retry := entry.Attempts < entry.MaxAttempts
		return s.outbox.MarkFailed(txCtx, entry.ID, errorMessage, retry)
	})
}

func (s *SyncService) handleSyncFailure(ctx context.Context, entry domain.OutboxEntry, message string, retry bool) error {
	err := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		logEntry := domain.SAPSyncLog{
			OrderItemID:  entry.OrderItemID,
			RFCFunction:  entry.RFCFunction,
			XMLRequest:   entry.XMLPayload,
			Status:       domain.SyncStatusFailed,
			ErrorMessage: &message,
		}
		if _, err := s.syncLogs.Create(txCtx, logEntry); err != nil {
			return err
		}

		itemStatus := domain.ItemStatusFailed
		if retry {
			itemStatus = domain.ItemStatusUpdated
		}
		if err := s.items.UpdateStatus(txCtx, entry.OrderItemID, itemStatus); err != nil {
			return err
		}

		return s.outbox.MarkFailed(txCtx, entry.ID, message, retry)
	})
	if err != nil {
		return err
	}

	if retry {
		return fmt.Errorf("%w: %s", domain.ErrSAPSyncFailed, message)
	}
	return nil
}

func (s *SyncService) ListLogs(ctx context.Context, orderID, itemID uuid.UUID) ([]domain.SAPSyncLog, error) {
	if _, err := s.items.GetByID(ctx, orderID, itemID); err != nil {
		return nil, err
	}
	return s.syncLogs.ListByItemID(ctx, itemID)
}

func (s *SyncService) GetLatestSyncStatus(ctx context.Context, orderID, itemID uuid.UUID) (*domain.SyncStatusResponse, error) {
	if _, err := s.items.GetByID(ctx, orderID, itemID); err != nil {
		return nil, err
	}

	entry, err := s.outbox.GetLatestByItemID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	return s.buildSyncStatusResponse(ctx, *entry)
}

func (s *SyncService) GetSyncStatus(ctx context.Context, orderID, itemID, outboxID uuid.UUID) (*domain.SyncStatusResponse, error) {
	if _, err := s.items.GetByID(ctx, orderID, itemID); err != nil {
		return nil, err
	}

	entry, err := s.outbox.GetByID(ctx, itemID, outboxID)
	if err != nil {
		return nil, err
	}

	return s.buildSyncStatusResponse(ctx, *entry)
}

func (s *SyncService) CancelSync(ctx context.Context, orderID, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error) {
	if _, err := s.items.GetByID(ctx, orderID, itemID); err != nil {
		return nil, err
	}

	var cancelled *domain.OutboxEntry
	err := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		entry, err := s.outbox.Cancel(txCtx, itemID, outboxID)
		if err != nil {
			return err
		}

		if err := s.items.UpdateStatus(txCtx, itemID, domain.ItemStatusPending); err != nil {
			return err
		}

		cancelled = entry
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.Info("sync cancelled",
		"order_id", orderID,
		"item_id", itemID,
		"outbox_id", outboxID,
	)

	return cancelled, nil
}

func (s *SyncService) buildSyncStatusResponse(ctx context.Context, entry domain.OutboxEntry) (*domain.SyncStatusResponse, error) {
	response := &domain.SyncStatusResponse{Outbox: entry}

	if entry.Status == domain.OutboxStatusCompleted || entry.Status == domain.OutboxStatusFailed {
		logs, err := s.syncLogs.ListByItemID(ctx, entry.OrderItemID)
		if err != nil {
			return nil, err
		}
		if len(logs) > 0 {
			response.LatestLog = &logs[0]
		}
	}

	return response, nil
}

func (s *SyncService) ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEntry, error) {
	return s.outbox.ClaimPending(ctx, limit)
}
