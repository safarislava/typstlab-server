package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/di"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/postgres"
)

func main() {
	cfg, err := config.Load("configs/config.json")
	if err != nil {
		slog.Error("Configuration error", slog.Any("error", err))
		os.Exit(1)
	}

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("Migration failure", slog.Any("error", err))
		os.Exit(1)
	}

	container := di.New(cfg)
	slog.Info("Starting server", slog.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, container.Router()); err != nil {
		slog.Error("Server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
