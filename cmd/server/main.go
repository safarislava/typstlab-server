package main

import (
	"log"
	"net/http"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/di"
	"github.com/safarislava/typstlab-server/internal/infrastructure/persistence/postgres"
)

func main() {
	cfg := config.Load("configs/config.json")

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Migration failure: %v", err)
	}

	container := di.New(cfg)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, container.Router()))
}
