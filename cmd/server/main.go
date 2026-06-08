package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/gofer/internal/delivery/http"
	"github.com/gofer/internal/infrastructure/postgres"
	"github.com/gofer/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const shutdownTimeout = 10 * time.Second

func main() {
	cfg := mustLoadConfig()
	pool := mustConnectDB(cfg)
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	router, hub := httpdelivery.NewRouter(pool, cfg)

	hubDone := make(chan struct{})
	go func() {
		defer close(hubDone)
		hub.Run(ctx)
	}()

	if err := runServer(ctx, router, cfg.Server.Port); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}

	select {
	case <-hubDone:
	case <-time.After(shutdownTimeout):
		slog.Warn("hub did not stop in time")
	}
	slog.Info("bye")
}

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}
	return cfg
}

func mustConnectDB(cfg *config.Config) *pgxpool.Pool {
	pool, err := postgres.NewPool(&cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)

	}
	return pool
}

func runServer(ctx context.Context, handler http.Handler, port string) error {
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	serverErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()
	slog.Info("server started", "port", port)

	select {
	case err, ok := <-serverErr:
		if ok && err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	slog.Info("http server stopped")
	return nil
}
