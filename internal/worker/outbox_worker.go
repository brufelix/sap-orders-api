package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/brufelix/sap-orders-api/internal/service"
)

type OutboxWorker struct {
	syncService *service.SyncService
	interval    time.Duration
	batchSize   int
	logger      *slog.Logger
}

func NewOutboxWorker(syncService *service.SyncService, interval time.Duration, batchSize int, logger *slog.Logger) *OutboxWorker {
	return &OutboxWorker{
		syncService: syncService,
		interval:    interval,
		batchSize:   batchSize,
		logger:      logger,
	}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("outbox worker started", "interval", w.interval, "batch_size", w.batchSize)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	entries, err := w.syncService.ClaimPending(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("failed to claim outbox entries", "error", err)
		return
	}

	for _, entry := range entries {
		if err := w.syncService.ProcessOutboxEntry(ctx, entry); err != nil {
			w.logger.Error("failed to process outbox entry",
				"outbox_id", entry.ID,
				"item_id", entry.OrderItemID,
				"error", err,
			)
			continue
		}

		w.logger.Info("outbox entry processed",
			"outbox_id", entry.ID,
			"item_id", entry.OrderItemID,
		)
	}
}
