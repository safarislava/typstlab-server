package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/application/auth"
	sessionApp "github.com/safarislava/typstlab-server/internal/application/session"
	"github.com/safarislava/typstlab-server/internal/application/user"
	"github.com/safarislava/typstlab-server/internal/domain/session"
	"github.com/safarislava/typstlab-server/internal/domain/token"
	domainUser "github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testRefreshToken = "valid_refresh_token"
)

type mockUserRepo struct {
	saveFunc        func(ctx context.Context, u *domainUser.User) error
	findByEmailFunc func(ctx context.Context, email string) (*domainUser.User, error)
	findByIDFunc    func(ctx context.Context, id uuid.UUID) (*domainUser.User, error)
}

func (m *mockUserRepo) Save(ctx context.Context, u *domainUser.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domainUser.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

type mockUserHasher struct {
	hashFunc    func(password string) (string, error)
	compareFunc func(hashedPassword, password string) error
}

func (m *mockUserHasher) Hash(password string) (string, error) {
	if m.hashFunc != nil {
		return m.hashFunc(password)
	}
	return "hashed_password", nil
}

func (m *mockUserHasher) Compare(hashedPassword, password string) error {
	if m.compareFunc != nil {
		return m.compareFunc(hashedPassword, password)
	}
	return nil
}

type mockUserTokenService struct {
	generateFunc func(userID uuid.UUID, role domainUser.Role) (token.Token, error)
	validateFunc func(t token.Token) (uuid.UUID, domainUser.Role, error)
}

func (m *mockUserTokenService) Generate(userID uuid.UUID, role domainUser.Role) (token.Token, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userID, role)
	}
	t, err := token.NewToken("mock-token")
	if err != nil {
		return token.Token{}, fmt.Errorf("failed to create domain token: %w", err)
	}
	return t, nil
}

func (m *mockUserTokenService) Validate(t token.Token) (uuid.UUID, domainUser.Role, error) {
	if m.validateFunc != nil {
		return m.validateFunc(t)
	}
	return uuid.Nil, domainUser.RoleGhost, errors.New("invalid")
}

type mockUserSessionRepo struct {
	saveFunc        func(ctx context.Context, s session.Session) error
	findByTokenFunc func(ctx context.Context, t token.Token) (session.Session, error)
	deleteFunc      func(ctx context.Context, t token.Token) error
}

func (m *mockUserSessionRepo) Save(ctx context.Context, s session.Session) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, s)
	}
	return nil
}

func (m *mockUserSessionRepo) FindByToken(ctx context.Context, t token.Token) (session.Session, error) {
	if m.findByTokenFunc != nil {
		return m.findByTokenFunc(ctx, t)
	}
	return session.Session{}, session.ErrSessionNotFound
}

func (m *mockUserSessionRepo) Delete(ctx context.Context, t token.Token) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, t)
	}
	return nil
}

func (m *mockUserSessionRepo) DeleteByUserID(context.Context, uuid.UUID) error {
	return nil
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	u, _ := domainUser.NewUser(uuid.New(), "login@example.com", "hashed_password", domainUser.RoleUser)
	repo := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domainUser.User, error) {
			if email == "login@example.com" {
				return u, nil
			}
			return nil, errors.New("not found")
		},
	}
	hasher := &mockUserHasher{}
	tokenSvc := &mockUserTokenService{}
	rtRepo := &mockUserSessionRepo{}
	svc := user.NewService(repo, hasher)
	rtSvc := sessionApp.NewService(rtRepo)
	authSvc := auth.NewService(svc, rtSvc, tokenSvc, hasher)
	handler := NewAuthHandler(authSvc)

	// Case 1: Invalid JSON
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString("{invalid-json"))
	rr := httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	// Case 2: Success
	body := `{"email":"login@example.com","password":"password"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body %s", rr.Code, rr.Body.String())
	}

	// Check that cookie was set
	rtCookie := getCookieByName(rr.Result().Cookies(), refreshTokenCookieName)
	if rtCookie == nil {
		t.Error("Expected refresh_token cookie to be set")
	} else if rtCookie.Value == "" {
		t.Error("Expected refresh token value to be non-empty")
	}

	// Case 3: Login failure
	body = `{"email":"wrong@example.com","password":"password"}`
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(body))
	rr = httptest.NewRecorder()
	handler.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	u, _ := domainUser.NewUser(userID, "login@example.com", "hashed_password", domainUser.RoleUser)
	rtVal, _ := token.NewToken(testRefreshToken)
	rt, _ := session.NewSession(rtVal, userID, time.Now().Add(1*time.Hour))

	repo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id uuid.UUID) (*domainUser.User, error) {
			return u, nil
		},
	}
	hasher := &mockUserHasher{}
	tokenSvc := &mockUserTokenService{}
	rtRepo := &mockUserSessionRepo{
		findByTokenFunc: func(ctx context.Context, t token.Token) (session.Session, error) {
			if t.Value() == testRefreshToken {
				return rt, nil
			}
			return session.Session{}, session.ErrSessionNotFound
		},
	}

	svc := user.NewService(repo, hasher)
	rtSvc := sessionApp.NewService(rtRepo)
	authSvc := auth.NewService(svc, rtSvc, tokenSvc, hasher)
	handler := NewAuthHandler(authSvc)

	// Case 1: Missing Cookie
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh", http.NoBody)
	rr := httptest.NewRecorder()
	handler.Refresh(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 when missing cookie, got %d", rr.Code)
	}

	// Case 2: Success
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh", http.NoBody)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: testRefreshToken})
	rr = httptest.NewRecorder()
	handler.Refresh(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body %s", rr.Code, rr.Body.String())
	}

	// Case 3: Invalid token in cookie
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh", http.NoBody)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "invalid"})
	rr = httptest.NewRecorder()
	handler.Refresh(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Parallel()

	repo := &mockUserRepo{}
	hasher := &mockUserHasher{}
	tokenSvc := &mockUserTokenService{}
	rtRepo := &mockUserSessionRepo{}
	svc := user.NewService(repo, hasher)
	rtSvc := sessionApp.NewService(rtRepo)
	authSvc := auth.NewService(svc, rtSvc, tokenSvc, hasher)
	handler := NewAuthHandler(authSvc)

	// Success logout
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/logout", http.NoBody)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "token"})
	rr := httptest.NewRecorder()
	handler.Logout(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}

	// Check that cookie was cleared (MaxAge < 0)
	rtCookie := getCookieByName(rr.Result().Cookies(), refreshTokenCookieName)
	if rtCookie == nil {
		t.Error("Expected refresh_token cookie to be returned")
	} else if rtCookie.MaxAge >= 0 {
		t.Errorf("Expected MaxAge to be negative to clear cookie, got %d", rtCookie.MaxAge)
	}
}

func getCookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
