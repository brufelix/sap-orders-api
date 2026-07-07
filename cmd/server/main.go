package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brufelix/sap-orders-api/internal/auth"
	"github.com/brufelix/sap-orders-api/internal/config"
	"github.com/brufelix/sap-orders-api/internal/handler"
	"github.com/brufelix/sap-orders-api/internal/repository"
	"github.com/brufelix/sap-orders-api/internal/sap"
	"github.com/brufelix/sap-orders-api/internal/service"
	"github.com/brufelix/sap-orders-api/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	authenticator, err := auth.NewAuthenticator(ctx, cfg.AzureTenantID, cfg.AzureAudience)
	if err != nil {
		logger.Error("auth setup failed", "error", err)
		os.Exit(1)
	}

	orderRepo := repository.NewOrderRepository(pool)
	itemRepo := repository.NewItemRepository(pool)
	syncRepo := repository.NewSyncRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	transactor := repository.NewTransactor(pool)

	var sapClient sap.Client = sap.NewStubClient(logger)

	orderService := service.NewOrderService(orderRepo, itemRepo)
	syncService := service.NewSyncService(orderRepo, itemRepo, syncRepo, outboxRepo, transactor, sapClient, cfg.SAPRFCFunction, logger)
	outboxWorker := worker.NewOutboxWorker(syncService, 5*time.Second, 10, logger)

	healthHandler := handler.NewHealthHandler(pool)
	orderHandler := handler.NewOrderHandler(orderService)
	syncHandler := handler.NewSyncHandler(syncService)

	tlsEnabled := cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(auth.SecurityHeaders)
	router.Use(auth.HTTPSRedirect(cfg.TLSRedirect || tlsEnabled))

	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)
	router.Get("/openapi.yaml", handler.OpenAPI)
	router.Get("/swagger", handler.SwaggerUI)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(httprate.LimitByIP(cfg.RateLimitRequests, time.Duration(cfg.RateLimitWindowSec)*time.Second))
		r.Use(handler.MaxBodyBytes(cfg.MaxBodyBytes))
		r.Use(authenticator.Middleware)

		r.With(auth.RequireScope(auth.ScopeOrdersRead)).Get("/orders", orderHandler.List)
		r.With(auth.RequireScope(auth.ScopeOrdersRead)).Get("/orders/{id}", orderHandler.Get)

		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Post("/orders", orderHandler.Create)
		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Patch("/orders/{id}", orderHandler.Update)
		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Post("/orders/{id}/items", orderHandler.AddItem)
		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Patch("/orders/{id}/items/{itemId}", orderHandler.UpdateItem)
		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Post("/orders/{id}/items/{itemId}/sync", syncHandler.SyncItem)
		r.With(auth.RequireScope(auth.ScopeOrdersWrite)).Delete("/orders/{id}/items/{itemId}/sync/{outboxId}", syncHandler.CancelSync)

		r.With(auth.RequireScope(auth.ScopeOrdersRead)).Get("/orders/{id}/items/{itemId}/sync", syncHandler.GetLatestStatus)
		r.With(auth.RequireScope(auth.ScopeOrdersRead)).Get("/orders/{id}/items/{itemId}/sync/{outboxId}", syncHandler.GetStatus)
		r.With(auth.RequireScope(auth.ScopeOrdersRead)).Get("/orders/{id}/items/{itemId}/sync-logs", syncHandler.ListLogs)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go outboxWorker.Run(ctx)

	go func() {
		if tlsEnabled {
			logger.Info("server listening with tls", "port", cfg.Port)
			if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
				os.Exit(1)
			}
			return
		}

		logger.Info("server listening", "port", cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("server stopped")
}
