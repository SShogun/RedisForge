package app

// app.go owns the lifecycle of every subsystem:
//   Open → Start workers → Serve HTTP → Shutdown sequence.
//
// Dependency injection: Currently manual (~100 lines) for clarity and visibility.
// At production scale (50+ components), consider google/wire to auto-generate this:
// https://github.com/google/wire
// For now, explicit wiring makes every dependency relationship explicit.

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SShogun/redisforge/internal/config"
	"github.com/SShogun/redisforge/internal/handlers"
	"github.com/SShogun/redisforge/internal/logging"
	"github.com/SShogun/redisforge/internal/observability"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/SShogun/redisforge/internal/workers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Run is the application entry point. Returns non-nil error on fatal startup failure.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("app.Run: load config: %w", err)
	}
	logger := logging.New(cfg.App)
	ctx := context.Background()
	otelShutdown, err := observability.InitTracer(ctx, "redisforge", cfg.App.Version)
	if err != nil {
		return fmt.Errorf("app.Run: init tracer: %w", err)
	}
	defer otelShutdown(context.Background())

	// ── Redis ──────────────────────────────────────────────────────
	redisClient, err := redisx.Open(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("app.Run: open redis: %w", err)
	}
	defer redisClient.Close()

	jsonStore := redisx.NewJSONStore(redisClient)
	bloomStore := redisx.NewBloomFilter(redisClient, "bf:idempotency")
	searchStore := redisx.NewSearchStore(redisClient)
	streamStore := redisx.NewStreamClient(redisClient)
	pubSubStore := redisx.NewPubSubClient(redisClient)
	_ = pubSubStore // used in handlers optionally

	// Initialise bloom filter (idempotent)
	if err := bloomStore.Reserve(ctx, 0.001, 1_000_000); err != nil {
		return fmt.Errorf("app.Run: bloom reserve: %w", err)
	}

	// Initialise search index (idempotent)
	if err := searchStore.EnsureIndex(ctx); err != nil {
		return fmt.Errorf("app.Run: ensure search index: %w", err)
	}

	// ── Repositories ───────────────────────────────────────────────
	memRepo := repo.NewMemoryItemRepo()
	cacheRepo := repo.NewCacheItemRepo(memRepo, jsonStore, logger)

	// ── Workers ────────────────────────────────────────────────────
	hostname, _ := os.Hostname()
	auditWorker := workers.NewAuditWorker(streamStore, logger, hostname)
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	if err := auditWorker.Start(workerCtx); err != nil {
		return fmt.Errorf("app.Run: start audit worker: %w", err)
	}

	// ── Router ─────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", handlers.HandleHealth())

	// root -> health for quick checks
	r.Get("/", handlers.HandleHealth())

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/v1/items", func(r chi.Router) {
		r.Post("/", handlers.HandleCreateItem(cacheRepo, streamStore, bloomStore))
		r.Get("/search", handlers.HandleSearchItems(searchStore))
		r.Get("/", handlers.HandleListItems(cacheRepo))
		r.Get("/{id}", handlers.HandleGetItem(cacheRepo))
		r.Put("/{id}", handlers.HandleUpdateItem(cacheRepo, streamStore))
		r.Delete("/{id}", handlers.HandleDeleteItem(cacheRepo))
	})

	// ── HTTP Server ────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// ── Graceful Shutdown ──────────────────────────────────────────
	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("redisforge listening", slog.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "err", err)
		}
	}()

	<-shutdownCtx.Done()
	logger.Info("shutdown signal received")

	// 1. Stop HTTP (finish in-flight requests)
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer httpCancel()
	_ = srv.Shutdown(httpCtx)
	logger.Info("http server stopped")

	// 2. Stop workers (finish current batch)
	cancelWorkers()
	auditWorker.Stop()
	logger.Info("workers stopped")

	// 3. Redis close is handled by defer above

	logger.Info("shutdown complete")
	return nil
}
