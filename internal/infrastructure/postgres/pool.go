package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofer/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	ctx := context.Background()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	lifetime, err := time.ParseDuration(cfg.MaxConnLifetime)
	if err != nil {
		return nil, fmt.Errorf("parse max_conn_lifetime: %w", err)
	}
	poolCfg.MaxConnLifetime = lifetime

	idleTime, err := time.ParseDuration(cfg.MaxConnIdleTime)
	if err != nil {
		return nil, fmt.Errorf("parse max_conn_idle_time: %w", err)
	}
	poolCfg.MaxConnIdleTime = idleTime

	healthCheck, err := time.ParseDuration(cfg.HealthCheckPeriod)
	if err != nil {
		return nil, fmt.Errorf("parse health_check_period: %w", err)
	}
	poolCfg.HealthCheckPeriod = healthCheck

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = pool.Ping(pingCtx)
	if err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("connected to database", "host", cfg.Host, "db", cfg.Name)

	return pool, nil
}
