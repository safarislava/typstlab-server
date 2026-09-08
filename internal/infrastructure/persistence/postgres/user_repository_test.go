package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

func TestNewUserRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewUserRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil UserRepository")
	}
}

func TestUserRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewUserRepository(pool)

	u, err := domain.NewUser(uuid.New(), "test@example.com", "hash123", domain.RoleUser)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if err := repo.Save(ctx, u); err == nil {
		t.Error("expected error saving with closed pool, got nil")
	}

	if _, err := repo.FindByEmail(ctx, "test@example.com"); err == nil {
		t.Error("expected error finding by email with closed pool, got nil")
	}

	if _, err := repo.FindByID(ctx, u.ID()); err == nil {
		t.Error("expected error finding by id with closed pool, got nil")
	}
}
