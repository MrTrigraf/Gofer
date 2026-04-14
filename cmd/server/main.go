package main

import (
	"context"
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
	runServer(router, cfg.Server.Port)
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

func runServer(handler http.Handler, port string) {
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		slog.Info("server started", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}
