package postgres

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	// Register PostgreSQL database driver for golang-migrate
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/safarislava/typstlab-server/migrations"
)

func RunMigrations(databaseURL string) error {
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver for migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("failed to close migration source driver", "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("failed to close migration database driver", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Database schema is up to date (no new migrations)")
			return nil
		}
		return fmt.Errorf("failed to apply database migrations: %w", err)
	}

	slog.Info("Database migrations applied successfully")
	return nil
}
