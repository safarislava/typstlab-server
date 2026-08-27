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

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/di"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/postgres"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("configs/config.json")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	container := di.New(cfg)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           container.Router(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	stopped := make(chan struct{})
	go func() {
		<-ctx.Done()
		shutdown(server)
		close(stopped)
	}()

	slog.Info("Server listening", slog.String("port", cfg.Port))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}

	<-stopped
	slog.Info("Server stopped cleanly")
	return nil
}

func shutdown(server *http.Server) {
	slog.Info("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", slog.Any("error", err))
	}
}
