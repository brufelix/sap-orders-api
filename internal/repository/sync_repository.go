package repository

import (
	"context"
	"fmt"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type syncRepository struct {
	pool *pgxpool.Pool
}

func NewSyncRepository(pool *pgxpool.Pool) SyncRepository {
	return &syncRepository{pool: pool}
}

func (r *syncRepository) Create(ctx context.Context, log domain.SAPSyncLog) (*domain.SAPSyncLog, error) {
	q := querier(ctx, r.pool)
	var created domain.SAPSyncLog
	err := q.QueryRow(ctx, `
		INSERT INTO sap_sync_logs (order_item_id, rfc_function, xml_request, xml_response, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, order_item_id, rfc_function, xml_request, xml_response, status, error_message, synced_at
	`, log.OrderItemID, log.RFCFunction, log.XMLRequest, log.XMLResponse, log.Status, log.ErrorMessage).Scan(
		&created.ID,
		&created.OrderItemID,
		&created.RFCFunction,
		&created.XMLRequest,
		&created.XMLResponse,
		&created.Status,
		&created.ErrorMessage,
		&created.SyncedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create sync log: %w", err)
	}

	return &created, nil
}

func (r *syncRepository) ListByItemID(ctx context.Context, itemID uuid.UUID) ([]domain.SAPSyncLog, error) {
	q := querier(ctx, r.pool)
	rows, err := q.Query(ctx, `
		SELECT id, order_item_id, rfc_function, xml_request, xml_response, status, error_message, synced_at
		FROM sap_sync_logs
		WHERE order_item_id = $1
		ORDER BY synced_at DESC
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("list sync logs: %w", err)
	}
	defer rows.Close()

	logs := make([]domain.SAPSyncLog, 0)
	for rows.Next() {
		var log domain.SAPSyncLog
		if err := rows.Scan(
			&log.ID,
			&log.OrderItemID,
			&log.RFCFunction,
			&log.XMLRequest,
			&log.XMLResponse,
			&log.Status,
			&log.ErrorMessage,
			&log.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sync log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}
