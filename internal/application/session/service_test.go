package session

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/session"
	tokenDomain "github.com/safarislava/typstlab-server/internal/domain/token"
)

const testTokenStr = "token-str"

type mockRepository struct {
	saveFunc           func(ctx context.Context, s domain.Session) error
	findByTokenFunc    func(ctx context.Context, t tokenDomain.Token) (domain.Session, error)
	deleteFunc         func(ctx context.Context, t tokenDomain.Token) error
	deleteByUserIDFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRepository) Save(ctx context.Context, s domain.Session) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) FindByToken(ctx context.Context, t tokenDomain.Token) (domain.Session, error) {
	if m.findByTokenFunc != nil {
		return m.findByTokenFunc(ctx, t)
	}
	return domain.Session{}, domain.ErrSessionNotFound
}

func (m *mockRepository) Delete(ctx context.Context, t tokenDomain.Token) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, t)
	}
	return nil
}

func (m *mockRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFunc != nil {
		return m.deleteByUserIDFunc(ctx, userID)
	}
	return nil
}

func TestService_Create_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{}
	svc := NewService(repo)

	userID := uuid.New()
	rt, err := svc.Create(context.Background(), userID, 1*time.Hour)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if rt.UserID() != userID {
		t.Errorf("Expected userID %s, got %s", userID, rt.UserID())
	}
	if rt.Token().Value() == "" {
		t.Error("Expected non-empty token value")
	}
}

func TestService_Get_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	rtVal, _ := tokenDomain.NewToken(testTokenStr)
	rt, _ := domain.NewSession(rtVal, userID, time.Now().Add(1*time.Hour))

	repo := &mockRepository{
		findByTokenFunc: func(ctx context.Context, t tokenDomain.Token) (domain.Session, error) {
			if t.Value() == testTokenStr {
				return rt, nil
			}
			return domain.Session{}, domain.ErrSessionNotFound
		},
	}
	svc := NewService(repo)

	found, err := svc.Get(context.Background(), rtVal)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if found.UserID() != userID {
		t.Errorf("Expected userID %s, got %s", userID, found.UserID())
	}
}

func TestService_Get_Error(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		findByTokenFunc: func(ctx context.Context, t tokenDomain.Token) (domain.Session, error) {
			return domain.Session{}, domain.ErrSessionNotFound
		},
	}
	svc := NewService(repo)

	rt, err := tokenDomain.NewToken("123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	_, err = svc.Get(context.Background(), rt)
	if err == nil {
		t.Fatalf("Expected error, got none")
	}
}

func TestService_Invalidate_Success(t *testing.T) {
	t.Parallel()

	deleted := false
	rtVal, _ := tokenDomain.NewToken(testTokenStr)
	repo := &mockRepository{
		deleteFunc: func(ctx context.Context, t tokenDomain.Token) error {
			if t.Value() == testTokenStr {
				deleted = true
			}
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.Invalidate(context.Background(), rtVal)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleted {
		t.Error("Expected token to be deleted")
	}
}

func TestService_Invalidate_Error(t *testing.T) {
	t.Parallel()

	repo := &mockRepository{
		deleteFunc: func(ctx context.Context, t tokenDomain.Token) error {
			return domain.ErrExpiredSession
		},
	}
	svc := NewService(repo)

	rt, err := tokenDomain.NewToken("123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	err = svc.Invalidate(context.Background(), rt)
	if err == nil {
		t.Fatalf("Expected error, got none")
	}
}
