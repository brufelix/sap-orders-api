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

type orderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{pool: pool}
}

func (r *orderRepository) List(ctx context.Context, filter domain.OrderListFilter) (*domain.PagedResult[domain.Order], error) {
	q := querier(ctx, r.pool)

	var total int64
	var statusFilter any
	if filter.Status != nil {
		statusFilter = string(*filter.Status)
	}

	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM orders
		WHERE ($1::text IS NULL OR status = $1)
	`, statusFilter).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count orders: %w", err)
	}

	offset := (filter.Page - 1) * filter.Limit
	rows, err := q.Query(ctx, `
		SELECT id, order_number, status, created_by, created_at, updated_at
		FROM orders
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, statusFilter, filter.Limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.Status,
			&order.CreatedBy,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return domain.NewPagedResult(orders, filter.Page, filter.Limit, total), nil
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	q := querier(ctx, r.pool)
	var order domain.Order
	err := q.QueryRow(ctx, `
		SELECT id, order_number, status, created_by, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.Status,
		&order.CreatedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	items, err := r.listItemsByOrderID(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return &order, nil
}

func (r *orderRepository) Create(ctx context.Context, orderNumber, createdBy string) (*domain.Order, error) {
	q := querier(ctx, r.pool)
	var order domain.Order
	err := q.QueryRow(ctx, `
		INSERT INTO orders (order_number, created_by)
		VALUES ($1, $2)
		RETURNING id, order_number, status, created_by, created_at, updated_at
	`, orderNumber, createdBy).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.Status,
		&order.CreatedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) Update(ctx context.Context, id uuid.UUID, status domain.OrderStatus) (*domain.Order, error) {
	q := querier(ctx, r.pool)
	var order domain.Order
	err := q.QueryRow(ctx, `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, order_number, status, created_by, created_at, updated_at
	`, id, status).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.Status,
		&order.CreatedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update order: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) listItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	q := querier(ctx, r.pool)
	rows, err := q.Query(ctx, `
		SELECT id, order_id, demand_code, description, delivery_date, status, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY delivery_date ASC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.DemandCode,
			&item.Description,
			&item.DeliveryDate,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
