package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/application/auth"
	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	"github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

type mockTokenService struct {
	validateFunc func(t token.Token) (uuid.UUID, domainUser.Role, error)
}

func (m *mockTokenService) Generate(uuid.UUID, domainUser.Role) (token.Token, error) {
	return token.Token{}, nil
}

func (m *mockTokenService) Validate(t token.Token) (uuid.UUID, domainUser.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(t)
	}
	return uuid.Nil, "", errors.New("invalid")
}

func setupTestMiddleware(tokenSvc *mockTokenService) *AuthMiddleware {
	userRepo := &mockUserRepo{}
	hasher := &mockUserHasher{}
	rtRepo := &mockUserSessionRepo{}

	userSvc := user.NewService(userRepo, hasher)
	rtSvc := sessionApp.NewService(rtRepo)
	authSvc := auth.NewService(userSvc, rtSvc, tokenSvc, hasher)

	return NewAuthMiddleware(authSvc)
}

func TestAuthMiddleware_Authenticate_NoAuthHeader(t *testing.T) {
	t.Parallel()

	tokenSvc := &mockTokenService{}
	mw := setupTestMiddleware(tokenSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Error("Expected userID context to be nil when no auth header is present")
	}
}

func TestAuthMiddleware_Authenticate_InvalidHeader(t *testing.T) {
	t.Parallel()

	tokenSvc := &mockTokenService{}
	mw := setupTestMiddleware(tokenSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "invalid")
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Error("Expected userID context to be nil when auth header is invalid")
	}
}

func TestAuthMiddleware_Authenticate_InvalidToken(t *testing.T) {
	t.Parallel()

	tokenSvc := &mockTokenService{
		validateFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			return uuid.Nil, "", errors.New("invalid token")
		},
	}
	mw := setupTestMiddleware(tokenSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid-token")
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Error("Expected userID context to be nil when token is invalid")
	}
}

func TestAuthMiddleware_Authenticate_ValidToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	role := domainUser.RoleUser

	tokenSvc := &mockTokenService{
		validateFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			if t.Value() == "valid-token" {
				return userID, role, nil
			}
			return uuid.Nil, "", errors.New("invalid token")
		},
	}
	mw := setupTestMiddleware(tokenSvc)

	var ctxUserID uuid.UUID
	var ctxRole domainUser.Role
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		ctxRole, _ = r.Context().Value(roleKey).(domainUser.Role)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid-token")
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != userID {
		t.Errorf("Expected userID context %s, got %s", userID, ctxUserID)
	}
	if ctxRole != role {
		t.Errorf("Expected role context %s, got %s", role, ctxRole)
	}
}

func TestRequireAuthenticated(t *testing.T) {
	t.Parallel()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Case 1: Unauthenticated
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	RequireAuthenticated(nextHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	// Case 2: Authenticated
	ctx := context.WithValue(context.Background(), userIDKey, uuid.New())
	req = httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	rr = httptest.NewRecorder()
	RequireAuthenticated(nextHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireRole(domainUser.RoleAdmin)

	// Case 1: No role
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	// Case 2: Disallowed role
	ctx := context.WithValue(context.Background(), roleKey, domainUser.RoleUser)
	req = httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	rr = httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", rr.Code)
	}

	// Case 3: Allowed role
	ctx = context.WithValue(context.Background(), roleKey, domainUser.RoleAdmin)
	req = httptest.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)
	rr = httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestUserIDFromContext(t *testing.T) {
	t.Parallel()

	// Case 1: Empty context
	_, ok := UserIDFromContext(context.Background())
	if ok {
		t.Error("Expected ok to be false from empty context, got true")
	}

	// Case 2: Valid context
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), userIDKey, userID)
	found, ok := UserIDFromContext(ctx)
	if !ok {
		t.Fatal("Expected ok to be true for valid context, got false")
	}
	if found != userID {
		t.Errorf("Expected userID %s, got %s", userID, found)
	}
}
