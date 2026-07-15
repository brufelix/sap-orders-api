package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) OutboxRepository {
	return &outboxRepository{pool: pool}
}

func (r *outboxRepository) Create(ctx context.Context, entry domain.OutboxEntry) (*domain.OutboxEntry, error) {
	q := querier(ctx, r.pool)
	var created domain.OutboxEntry
	err := q.QueryRow(ctx, `
		INSERT INTO sap_outbox (order_item_id, order_number, rfc_function, xml_payload, status, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, order_item_id, order_number, rfc_function, xml_payload, status, attempts, max_attempts, error_message, created_at, processed_at
	`, entry.OrderItemID, entry.OrderNumber, entry.RFCFunction, entry.XMLPayload, entry.Status, entry.MaxAttempts).Scan(
		&created.ID,
		&created.OrderItemID,
		&created.OrderNumber,
		&created.RFCFunction,
		&created.XMLPayload,
		&created.Status,
		&created.Attempts,
		&created.MaxAttempts,
		&created.ErrorMessage,
		&created.CreatedAt,
		&created.ProcessedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create outbox entry: %w", err)
	}

	return &created, nil
}

func (r *outboxRepository) GetByID(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error) {
	q := querier(ctx, r.pool)
	var entry domain.OutboxEntry
	err := q.QueryRow(ctx, `
		SELECT id, order_item_id, order_number, rfc_function, xml_payload, status, attempts, max_attempts, error_message, created_at, processed_at
		FROM sap_outbox
		WHERE id = $1 AND order_item_id = $2
	`, outboxID, itemID).Scan(
		&entry.ID,
		&entry.OrderItemID,
		&entry.OrderNumber,
		&entry.RFCFunction,
		&entry.XMLPayload,
		&entry.Status,
		&entry.Attempts,
		&entry.MaxAttempts,
		&entry.ErrorMessage,
		&entry.CreatedAt,
		&entry.ProcessedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get outbox entry: %w", err)
	}

	return &entry, nil
}

func (r *outboxRepository) GetLatestByItemID(ctx context.Context, itemID uuid.UUID) (*domain.OutboxEntry, error) {
	q := querier(ctx, r.pool)
	var entry domain.OutboxEntry
	err := q.QueryRow(ctx, `
		SELECT id, order_item_id, order_number, rfc_function, xml_payload, status, attempts, max_attempts, error_message, created_at, processed_at
		FROM sap_outbox
		WHERE order_item_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, itemID).Scan(
		&entry.ID,
		&entry.OrderItemID,
		&entry.OrderNumber,
		&entry.RFCFunction,
		&entry.XMLPayload,
		&entry.Status,
		&entry.Attempts,
		&entry.MaxAttempts,
		&entry.ErrorMessage,
		&entry.CreatedAt,
		&entry.ProcessedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get latest outbox entry: %w", err)
	}

	return &entry, nil
}

func (r *outboxRepository) HasActiveByItemID(ctx context.Context, itemID uuid.UUID) (bool, error) {
	q := querier(ctx, r.pool)
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM sap_outbox
			WHERE order_item_id = $1 AND status IN ('PENDING', 'PROCESSING')
		)
	`, itemID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active outbox: %w", err)
	}
	return exists, nil
}

func (r *outboxRepository) Cancel(ctx context.Context, itemID, outboxID uuid.UUID) (*domain.OutboxEntry, error) {
	q := querier(ctx, r.pool)
	var entry domain.OutboxEntry
	err := q.QueryRow(ctx, `
		UPDATE sap_outbox
		SET status = 'CANCELLED', processed_at = NOW(), error_message = 'cancelled by user'
		WHERE id = $1 AND order_item_id = $2 AND status = 'PENDING'
		RETURNING id, order_item_id, order_number, rfc_function, xml_payload, status, attempts, max_attempts, error_message, created_at, processed_at
	`, outboxID, itemID).Scan(
		&entry.ID,
		&entry.OrderItemID,
		&entry.OrderNumber,
		&entry.RFCFunction,
		&entry.XMLPayload,
		&entry.Status,
		&entry.Attempts,
		&entry.MaxAttempts,
		&entry.ErrorMessage,
		&entry.CreatedAt,
		&entry.ProcessedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := r.GetByID(ctx, itemID, outboxID)
		if getErr != nil {
			return nil, getErr
		}
		if existing.Status != domain.OutboxStatusPending {
			return nil, domain.ErrSyncNotCancellable
		}
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cancel outbox entry: %w", err)
	}

	return &entry, nil
}

func (r *outboxRepository) ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEntry, error) {
	q := querier(ctx, r.pool)
	rows, err := q.Query(ctx, `
		UPDATE sap_outbox
		SET status = 'PROCESSING', attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM sap_outbox
			WHERE status = 'PENDING' AND attempts < max_attempts
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, order_item_id, order_number, rfc_function, xml_payload, status, attempts, max_attempts, error_message, created_at, processed_at
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending outbox entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.OutboxEntry, 0)
	for rows.Next() {
		var entry domain.OutboxEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.OrderItemID,
			&entry.OrderNumber,
			&entry.RFCFunction,
			&entry.XMLPayload,
			&entry.Status,
			&entry.Attempts,
			&entry.MaxAttempts,
			&entry.ErrorMessage,
			&entry.CreatedAt,
			&entry.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *outboxRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	q := querier(ctx, r.pool)
	_, err := q.Exec(ctx, `
		UPDATE sap_outbox
		SET status = 'COMPLETED', processed_at = NOW(), error_message = NULL
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("mark outbox completed: %w", err)
	}
	return nil
}

func (r *outboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string, retry bool) error {
	q := querier(ctx, r.pool)
	status := domain.OutboxStatusFailed
	if retry {
		status = domain.OutboxStatusPending
	}

	_, err := q.Exec(ctx, `
		UPDATE sap_outbox
		SET status = $2, error_message = $3, processed_at = CASE WHEN $4 THEN NULL ELSE NOW() END
		WHERE id = $1
	`, id, status, errorMessage, retry)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}
