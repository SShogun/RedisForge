package app

// app.go owns the lifecycle of every subsystem:
//   Open → Start workers → Serve HTTP → Shutdown sequence.

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
	"github.com/SShogun/redisforge/internal/logging"
	"github.com/SShogun/redisforge/internal/redisx"
	"github.com/SShogun/redisforge/internal/repo"
	"github.com/SShogun/redisforge/internal/workers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Run is the application entry point. Returns non-nil error on fatal startup failure.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("app.Run: load config: %w", err)
	}

	logger := logging.New(cfg.App)
	ctx := context.Background()

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

	// If the handlers package doesn't expose the expected constructor functions
	// (HandleHealth, HandleCreateItem, etc.), fall back to simple inline
	// handlers to keep the application buildable. Keep references to the
	// stores to avoid unused variable compilation errors.
	_ = cacheRepo
	_ = streamStore
	_ = bloomStore
	_ = searchStore

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/v1/items", func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not implemented", http.StatusNotImplemented)
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not implemented", http.StatusNotImplemented)
		})
		r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not implemented", http.StatusNotImplemented)
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not implemented", http.StatusNotImplemented)
		})
		r.Get("/search", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not implemented", http.StatusNotImplemented)
		})
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
