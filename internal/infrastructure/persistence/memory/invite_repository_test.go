package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/invite"
)

func TestMemoryInviteRepository_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewMemoryInviteRepository()
	ctx := context.Background()

	// Non-existent by code
	if _, err := repo.FindByCode(ctx, "nonexistent"); err == nil {
		t.Error("Expected error for non-existent invite code")
	}

	// Non-existent by ID
	if _, err := repo.FindByID(ctx, uuid.New()); err == nil {
		t.Error("Expected error for non-existent invite id")
	}
}

func TestMemoryInviteRepository_CRUD(t *testing.T) {
	t.Parallel()

	repo := NewMemoryInviteRepository()
	ctx := context.Background()

	invID := uuid.New()
	projectID := uuid.New()
	createdBy := uuid.New()
	code := "test-code-xyz"
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	inv, err := domain.NewInvite(invID, projectID, createdBy, code, exp, now)
	if err != nil {
		t.Fatalf("Failed to create domain invite: %v", err)
	}

	if err := repo.Save(ctx, inv); err != nil {
		t.Fatalf("Failed to save invite: %v", err)
	}

	// Find by code
	foundByCode, err := repo.FindByCode(ctx, code)
	if err != nil || foundByCode.ID() != invID {
		t.Errorf("FindByCode mismatch: %v, err: %v", foundByCode, err)
	}

	// Find by ID
	foundByID, err := repo.FindByID(ctx, invID)
	if err != nil || foundByID.Code() != code {
		t.Errorf("FindByID mismatch: %v, err: %v", foundByID, err)
	}

	// Find by Project ID
	invites, err := repo.FindByProjectID(ctx, projectID)
	if err != nil || len(invites) != 1 {
		t.Errorf("FindByProjectID expected 1 invite, got %d, err: %v", len(invites), err)
	}

	// Delete
	if err := repo.Delete(ctx, invID); err != nil {
		t.Fatalf("Failed to delete invite: %v", err)
	}
	if _, err := repo.FindByID(ctx, invID); err == nil {
		t.Error("Expected invite to be deleted")
	}
}
