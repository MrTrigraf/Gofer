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

	httpdelivery "github.com/gofer/internal/delivery/http"
	"github.com/gofer/internal/infrastructure/postgres"
	"github.com/gofer/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := mustLoadConfig()
	pool := mustConnectDB(cfg)
	defer pool.Close()

	router, hub := httpdelivery.NewRouter(pool, cfg)
	go hub.Run()

	if err := runServer(router, cfg.Server.Port); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
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

func startServer(server *http.Server) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	return errCh
}

func waitSignal() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

func runServer(handler http.Handler, port string) error {
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	errCh := startServer(server)
	slog.Info("server started", "port", port)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-waitSignal():
	}

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
