package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

func TestMemoryProjectRepository(t *testing.T) {
	t.Parallel()

	repo := NewMemoryProjectRepository()
	ctx := context.Background()

	// Test FindByID non-existent project (passing uuid.UUID directly)
	_, err := repo.FindByID(ctx, uuid.New())
	if err == nil {
		t.Error("Expected error when searching for non-existent project, got nil")
	}

	// Create a valid project
	p, err := domain.NewProject(uuid.New(), []uuid.UUID{uuid.New()}, "Alpha", time.Now())
	if err != nil {
		t.Fatalf("Failed to create domain project: %v", err)
	}

	// Save project
	err = repo.Save(ctx, p)
	if err != nil {
		t.Fatalf("Failed to save project: %v", err)
	}

	// Find project by ID (passing uuid.UUID directly)
	found, err := repo.FindByID(ctx, p.ID())
	if err != nil {
		t.Fatalf("Failed to find saved project: %v", err)
	}

	if found.ID() != p.ID() || found.Name() != p.Name() {
		t.Errorf("Found project properties do not match: got %+v, want %+v", found, p)
	}
}
