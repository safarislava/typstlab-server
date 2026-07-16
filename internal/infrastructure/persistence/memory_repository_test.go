package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	projectDomain "github.com/safarislava/typstlab-server/internal/domain/project"
	userDomain "github.com/safarislava/typstlab-server/internal/domain/user"
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
	p, err := projectDomain.NewProject(uuid.New(), []uuid.UUID{uuid.New()}, "Alpha", time.Now())
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

func TestMemoryUserRepository(t *testing.T) {
	t.Parallel()

	repo := NewMemoryUserRepository()
	ctx := context.Background()

	userID := uuid.New()
	email := "test@example.com"
	u, err := userDomain.NewUser(userID, email, "hashed_password", userDomain.RoleUser)
	if err != nil {
		t.Fatalf("Failed to create domain user: %v", err)
	}

	// 1. Find by non-existent email
	_, err = repo.FindByEmail(ctx, email)
	if err == nil {
		t.Error("Expected error finding non-existent user by email, got nil")
	}

	// 2. Find by non-existent ID
	_, err = repo.FindByID(ctx, userID)
	if err == nil {
		t.Error("Expected error finding non-existent user by ID, got nil")
	}

	// 3. Save user
	err = repo.Save(ctx, u)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// 4. Find by email (success)
	foundByEmail, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to find user by email: %v", err)
	}
	if foundByEmail.ID() != userID {
		t.Errorf("Expected user ID %s, got %s", userID, foundByEmail.ID())
	}

	// 5. Find by ID (success)
	foundByID, err := repo.FindByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to find user by ID: %v", err)
	}
	if foundByID.Email() != email {
		t.Errorf("Expected user email %s, got %s", email, foundByID.Email())
	}
}
