package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

const testValidCookieToken = "valid-cookie-token"

type mockAuthService struct {
	authorizeFunc func(t token.Token) (uuid.UUID, domainUser.Role, error)
}

func (m *mockAuthService) Authorize(t token.Token) (uuid.UUID, domainUser.Role, error) {
	if m.authorizeFunc != nil {
		return m.authorizeFunc(t)
	}
	return uuid.Nil, "", errors.New("unauthorized")
}

func TestAuthMiddleware_Authenticate_NoAuthHeader(t *testing.T) {
	t.Parallel()

	authSvc := &mockAuthService{}
	mw := NewAuthMiddleware(authSvc)

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

func TestAuthMiddleware_Authenticate_AuthorizationHeaderIgnored(t *testing.T) {
	t.Parallel()

	authSvc := &mockAuthService{
		authorizeFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			return uuid.Nil, "", errors.New("should not be called for Authorization header")
		},
	}
	mw := NewAuthMiddleware(authSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("Authorization", "Bearer valid-token")
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Error("Expected userID context to be nil when token is in Authorization header")
	}
}

func TestAuthMiddleware_Authenticate_InvalidToken(t *testing.T) {
	t.Parallel()

	authSvc := &mockAuthService{
		authorizeFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			return uuid.Nil, "", errors.New("invalid token")
		},
	}
	mw := NewAuthMiddleware(authSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: AccessTokenCookieName, Value: "invalid-token"})
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Error("Expected userID context to be nil when token is invalid")
	}
}

func TestAuthMiddleware_Authenticate_ValidTokenInCookie(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	role := domainUser.RoleUser

	authSvc := &mockAuthService{
		authorizeFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			if t.Value() == testValidCookieToken {
				return userID, role, nil
			}
			return uuid.Nil, "", errors.New("invalid token")
		},
	}
	mw := NewAuthMiddleware(authSvc)

	var ctxUserID uuid.UUID
	var ctxRole domainUser.Role
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		ctxRole, _ = r.Context().Value(roleKey).(domainUser.Role)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws/projects/123", http.NoBody)
	req.AddCookie(&http.Cookie{Name: AccessTokenCookieName, Value: testValidCookieToken})
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != userID {
		t.Errorf("Expected userID context %s, got %s", userID, ctxUserID)
	}
	if ctxRole != role {
		t.Errorf("Expected role context %s, got %s", role, ctxRole)
	}
}

func TestAuthMiddleware_Authenticate_QueryParamIgnored(t *testing.T) {
	t.Parallel()

	authSvc := &mockAuthService{
		authorizeFunc: func(t token.Token) (uuid.UUID, domainUser.Role, error) {
			return uuid.Nil, "", errors.New("should not be called for query params")
		},
	}
	mw := NewAuthMiddleware(authSvc)

	var ctxUserID uuid.UUID
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, _ = r.Context().Value(userIDKey).(uuid.UUID)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws/projects/123?token=some-token", http.NoBody)
	mw.Authenticate(nextHandler).ServeHTTP(httptest.NewRecorder(), req)
	if ctxUserID != uuid.Nil {
		t.Errorf("Expected userID context to be nil when token is only in query param, got %s", ctxUserID)
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

func TestWithUserID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := WithUserID(context.Background(), userID)
	found, ok := UserIDFromContext(ctx)
	if !ok || found != userID {
		t.Errorf("Expected %s, got %s (ok=%v)", userID, found, ok)
	}
}
