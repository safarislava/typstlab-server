package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/session"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
)

func TestNewSessionRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	repo := NewSessionRepository(pool)
	if repo == nil {
		t.Fatal("expected non-nil SessionRepository")
	}
}

func TestSessionRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, err := NewPool(ctx, testValidDatabaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	pool.Close()

	repo := NewSessionRepository(pool)

	tok, err := domainToken.NewToken("test-token-12345678901234567890")
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	sess, err := domain.NewSession(tok, uuid.New(), time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := repo.Save(ctx, sess); err == nil {
		t.Error("expected error saving session with closed pool, got nil")
	}

	if _, err := repo.FindByToken(ctx, tok); err == nil {
		t.Error("expected error finding session with closed pool, got nil")
	}

	if err := repo.Delete(ctx, tok); err == nil {
		t.Error("expected error deleting session with closed pool, got nil")
	}

	if err := repo.DeleteByUserID(ctx, sess.UserID()); err == nil {
		t.Error("expected error deleting sessions by user id with closed pool, got nil")
	}
}
