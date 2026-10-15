package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/invite"
)

func TestNewInviteRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewInviteRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil InviteRepository")
	}
}

func TestInviteRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewInviteRepository(pool)

	inv, err := domain.NewInvite(uuid.New(), uuid.New(), uuid.New(), "code123", time.Now().Add(24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("failed to create invite: %v", err)
	}

	if err := repo.Save(ctx, inv); err == nil {
		t.Error("expected error saving invite with closed pool, got nil")
	}

	if _, err := repo.FindByCode(ctx, inv.Code()); err == nil {
		t.Error("expected error finding invite by code with closed pool, got nil")
	}

	if _, err := repo.FindByProjectID(ctx, inv.ProjectID()); err == nil {
		t.Error("expected error finding invites by project id with closed pool, got nil")
	}

	if _, err := repo.FindByID(ctx, inv.ID()); err == nil {
		t.Error("expected error finding invite by id with closed pool, got nil")
	}

	if err := repo.Delete(ctx, inv.ID()); err == nil {
		t.Error("expected error deleting invite with closed pool, got nil")
	}
}
