package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string, maxAttempts int, retryDelay time.Duration, maxConns int32, logger *slog.Logger) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("DSN de PostgreSQL inválido: %w", err)
	}
	cfg.MaxConns = maxConns

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return pool, nil
			} else {
				lastErr = pingErr
				pool.Close()
			}
		} else {
			lastErr = err
		}

		logger.Warn("PostgreSQL connection attempt failed", "attempt", attempt, "error", lastErr.Error())
		if attempt < maxAttempts {
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}
	return nil, fmt.Errorf("no fue posible conectar a PostgreSQL: %w", lastErr)
}
