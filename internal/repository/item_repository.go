package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type itemRepository struct {
	pool *pgxpool.Pool
}

func NewItemRepository(pool *pgxpool.Pool) ItemRepository {
	return &itemRepository{pool: pool}
}

func (r *itemRepository) Create(ctx context.Context, orderID uuid.UUID, demandCode, description string, deliveryDate time.Time) (*domain.OrderItem, error) {
	q := querier(ctx, r.pool)
	var item domain.OrderItem
	err := q.QueryRow(ctx, `
		INSERT INTO order_items (order_id, demand_code, description, delivery_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, order_id, demand_code, description, delivery_date, status, created_at, updated_at
	`, orderID, demandCode, description, deliveryDate).Scan(
		&item.ID,
		&item.OrderID,
		&item.DemandCode,
		&item.Description,
		&item.DeliveryDate,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create order item: %w", err)
	}

	return &item, nil
}

func (r *itemRepository) GetByID(ctx context.Context, orderID, itemID uuid.UUID) (*domain.OrderItem, error) {
	q := querier(ctx, r.pool)
	var item domain.OrderItem
	err := q.QueryRow(ctx, `
		SELECT id, order_id, demand_code, description, delivery_date, status, created_at, updated_at
		FROM order_items
		WHERE id = $1 AND order_id = $2
	`, itemID, orderID).Scan(
		&item.ID,
		&item.OrderID,
		&item.DemandCode,
		&item.Description,
		&item.DeliveryDate,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order item: %w", err)
	}

	return &item, nil
}

func (r *itemRepository) Update(ctx context.Context, orderID, itemID uuid.UUID, description *string, deliveryDate *time.Time, status *domain.ItemStatus) (*domain.OrderItem, error) {
	current, err := r.GetByID(ctx, orderID, itemID)
	if err != nil {
		return nil, err
	}

	if description != nil {
		current.Description = *description
	}
	if deliveryDate != nil {
		current.DeliveryDate = *deliveryDate
	}
	if status != nil {
		current.Status = *status
	}

	q := querier(ctx, r.pool)
	var item domain.OrderItem
	err = q.QueryRow(ctx, `
		UPDATE order_items
		SET description = $3, delivery_date = $4, status = $5, updated_at = NOW()
		WHERE id = $1 AND order_id = $2
		RETURNING id, order_id, demand_code, description, delivery_date, status, created_at, updated_at
	`, itemID, orderID, current.Description, current.DeliveryDate, current.Status).Scan(
		&item.ID,
		&item.OrderID,
		&item.DemandCode,
		&item.Description,
		&item.DeliveryDate,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update order item: %w", err)
	}

	return &item, nil
}

func (r *itemRepository) UpdateStatus(ctx context.Context, itemID uuid.UUID, status domain.ItemStatus) error {
	q := querier(ctx, r.pool)
	tag, err := q.Exec(ctx, `
		UPDATE order_items
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, itemID, status)
	if err != nil {
		return fmt.Errorf("update item status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
