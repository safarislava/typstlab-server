package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

func TestNewProjectRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewProjectRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil ProjectRepository")
	}
}

func TestProjectRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewProjectRepository(pool)

	proj, err := domain.NewProject(uuid.New(), []uuid.UUID{uuid.New()}, "Test Project", time.Now())
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := repo.Save(ctx, proj); err == nil {
		t.Error("expected error saving project with closed pool, got nil")
	}

	if _, err := repo.FindByID(ctx, proj.ID()); err == nil {
		t.Error("expected error finding project with closed pool, got nil")
	}

	if err := repo.Delete(ctx, proj.ID()); err == nil {
		t.Error("expected error deleting project with closed pool, got nil")
	}
}
