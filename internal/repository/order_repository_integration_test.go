//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
	"github.com/brufelix/sap-orders-api/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestOrderRepository_CreateAndList_Integration(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewOrderRepository(pool)

	ctx := context.Background()
	created, err := repo.Create(ctx, "PO-INT-001", "integration@test.com")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if got.OrderNumber != "PO-INT-001" {
		t.Fatalf("expected PO-INT-001, got %s", got.OrderNumber)
	}

	result, err := repo.List(ctx, domain.OrderListFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if result.Total < 1 {
		t.Fatalf("expected at least one order, got total=%d", result.Total)
	}
}

func TestOrderRepository_ListPagination_Integration(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewOrderRepository(pool)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if _, err := repo.Create(ctx, fmt.Sprintf("PO-PAGE-%03d", i), "integration@test.com"); err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
	}

	result, err := repo.List(ctx, domain.OrderListFilter{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 items on page, got %d", len(result.Data))
	}
	if result.Total < 3 {
		t.Fatalf("expected total >= 3, got %d", result.Total)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("saporders"),
		postgres.WithUsername("saporders"),
		postgres.WithPassword("saporders"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	pool, err := repository.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	if err := applyMigrations(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return pool
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return err
		}
	}

	return nil
}

func TestOrderRepository_GetByID_NotFound_Integration(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewOrderRepository(pool)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
