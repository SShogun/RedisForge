package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/SShogun/redisforge/internal/config"
)

type contextKey string

const loggerKey contextKey = "logger"

func New(cfg config.App) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}

	var handler slog.Handler
	if cfg.Env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("service", "redisforge"),
		slog.String("env", cfg.Env),
		slog.String("version", cfg.Version),
	)
}

func WithRequestID(ctx context.Context, logger *slog.Logger, id string) context.Context {
	enriched := logger.With(slog.String("request_id", id))
	return context.WithValue(ctx, loggerKey, enriched)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
