package postgres

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/safarislava/typstlab-server/migrations"
)

func TestMigrationsFS(t *testing.T) {
	t.Parallel()

	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatalf("Failed to initialize iofs driver with embedded migrations: %v", err)
	}

	firstVersion, err := d.First()
	if err != nil {
		t.Fatalf("Failed to get first migration version: %v", err)
	}

	if firstVersion != 1 {
		t.Errorf("Expected first migration version to be 1, got %d", firstVersion)
	}
}
