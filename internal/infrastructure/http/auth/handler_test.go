package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	appAuth "github.com/safarislava/typstlab-server/internal/application/auth"
	"github.com/safarislava/typstlab-server/internal/domain/session"
	"github.com/safarislava/typstlab-server/internal/domain/token"
)

const (
	testRefreshToken = "valid_refresh_token"
)

type mockAuthService struct {
	loginFunc   func(ctx context.Context, req appAuth.LoginRequest) (*appAuth.LoginResponse, error)
	refreshFunc func(ctx context.Context, req appAuth.RefreshRequest) (*appAuth.RefreshResponse, error)
	logoutFunc  func(ctx context.Context, refreshToken token.Token) error
}

func (m *mockAuthService) Login(ctx context.Context, req appAuth.LoginRequest) (*appAuth.LoginResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, req)
	}
	return nil, errors.New("login failed")
}

func (m *mockAuthService) Refresh(ctx context.Context, req appAuth.RefreshRequest) (*appAuth.RefreshResponse, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(ctx, req)
	}
	return nil, errors.New("refresh failed")
}

func (m *mockAuthService) Logout(ctx context.Context, refreshToken token.Token) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, refreshToken)
	}
	return nil
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	at, _ := token.NewToken("access_token")
	rtVal, _ := token.NewToken(testRefreshToken)
	s, _ := session.NewSession(rtVal, uuid.New(), time.Now().Add(24*time.Hour))

	authSvc := &mockAuthService{
		loginFunc: func(ctx context.Context, req appAuth.LoginRequest) (*appAuth.LoginResponse, error) {
			if req.Email == "login@example.com" && req.Password == "password" {
				return &appAuth.LoginResponse{
					AccessToken:  at,
					RefreshToken: s,
				}, nil
			}
			return nil, errors.New("unauthorized")
		},
	}
	handler := NewHandler(authSvc)

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

	at, _ := token.NewToken("access_token")
	rtVal, _ := token.NewToken(testRefreshToken)
	s, _ := session.NewSession(rtVal, uuid.New(), time.Now().Add(24*time.Hour))

	authSvc := &mockAuthService{
		refreshFunc: func(ctx context.Context, req appAuth.RefreshRequest) (*appAuth.RefreshResponse, error) {
			if req.RefreshToken.Value() == testRefreshToken {
				return &appAuth.RefreshResponse{
					AccessToken:  at,
					RefreshToken: s,
				}, nil
			}
			return nil, errors.New("invalid refresh token")
		},
	}
	handler := NewHandler(authSvc)

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

	logoutCalled := false
	authSvc := &mockAuthService{
		logoutFunc: func(ctx context.Context, rt token.Token) error {
			logoutCalled = true
			return nil
		},
	}
	handler := NewHandler(authSvc)

	// Success logout
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/logout", http.NoBody)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "token"})
	rr := httptest.NewRecorder()
	handler.Logout(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}

	if !logoutCalled {
		t.Error("Expected logout to be called")
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
