package di

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
)

const (
	testDatabaseURL = "postgres://user:pass@localhost:5432/db"
	testJWTSecret   = "test-secret"
	testOrigin      = "http://localhost:3000"
	testAuthJSON    = `{"email":"di_user@example.com","password":"password123"}`
)

func newTestConfig() *config.Config {
	return &config.Config{
		Port:           "8080",
		DatabaseURL:    testDatabaseURL,
		JWTSecret:      testJWTSecret,
		AllowedOrigins: []string{testOrigin},
	}
}

func assertSingleton[T comparable](t *testing.T, name string, first, second T) {
	t.Helper()
	var zero T
	if first == zero || first != second {
		t.Errorf("%s should return non-nil singleton instance", name)
	}
}

func assertNil[T comparable](t *testing.T, name string, val T) {
	t.Helper()
	var zero T
	if val != zero {
		t.Fatalf("%s should be nil initially", name)
	}
}

func TestContainer_InitialStateIsNil(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertNil(t, "projectRepo", c.projectRepo)
	assertNil(t, "fileRepo", c.fileRepo)
	assertNil(t, "userRepo", c.userRepo)
	assertNil(t, "sessionRepo", c.sessionRepo)

	assertNil(t, "hasher", c.hasher)
	assertNil(t, "tokenService", c.tokenService)
	assertNil(t, "yjsMerger", c.yjsMerger)

	assertNil(t, "projectService", c.projectService)
	assertNil(t, "fileService", c.fileService)
	assertNil(t, "syncService", c.syncService)
	assertNil(t, "userService", c.userService)
	assertNil(t, "sessionService", c.sessionService)
	assertNil(t, "authService", c.authService)

	assertNil(t, "projectHandler", c.projectHandler)
	assertNil(t, "fileHandler", c.fileHandler)
	assertNil(t, "syncHandler", c.syncHandler)
	assertNil(t, "userHandler", c.userHandler)
	assertNil(t, "authHandler", c.authHandler)
	assertNil(t, "authMiddleware", c.authMiddleware)
	assertNil(t, "accessMiddleware", c.accessMiddleware)

	assertNil(t, "router", c.router)
}

func TestContainer_Config(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig()
	c := New(cfg)

	if c.Config() != cfg {
		t.Errorf("Expected config %v, got %v", cfg, c.Config())
	}
}

func TestContainer_Repositories(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertSingleton(t, "ProjectRepo", c.ProjectRepo(), c.ProjectRepo())
	assertSingleton(t, "FileRepo", c.FileRepo(), c.FileRepo())
	assertSingleton(t, "UserRepo", c.UserRepo(), c.UserRepo())
	assertSingleton(t, "SessionRepo", c.SessionRepo(), c.SessionRepo())
}

func TestContainer_Infrastructure(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertSingleton(t, "Hasher", c.Hasher(), c.Hasher())
	assertSingleton(t, "TokenService", c.TokenService(), c.TokenService())
	assertSingleton(t, "YjsMerger", c.YjsMerger(), c.YjsMerger())
}

func TestContainer_Services(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertSingleton(t, "ProjectService", c.ProjectService(), c.ProjectService())
	assertSingleton(t, "FileService", c.FileService(), c.FileService())
	assertSingleton(t, "EntryService", c.EntryService(), c.EntryService())
	assertSingleton(t, "MetadataService", c.MetadataService(), c.MetadataService())
	assertSingleton(t, "SyncMetadataService", c.SyncMetadataService(), c.SyncMetadataService())
	assertSingleton(t, "SyncFileService", c.SyncFileService(), c.SyncFileService())
	assertSingleton(t, "SyncService", c.SyncService(), c.SyncService())
	assertSingleton(t, "UserService", c.UserService(), c.UserService())
	assertSingleton(t, "SessionService", c.SessionService(), c.SessionService())
	assertSingleton(t, "AuthService", c.AuthService(), c.AuthService())
}

func TestContainer_HandlersAndMiddlewares(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertSingleton(t, "ProjectHandler", c.ProjectHandler(), c.ProjectHandler())
	assertSingleton(t, "FileHandler", c.FileHandler(), c.FileHandler())
	assertSingleton(t, "SyncHandler", c.SyncHandler(), c.SyncHandler())
	assertSingleton(t, "UserHandler", c.UserHandler(), c.UserHandler())
	assertSingleton(t, "AuthHandler", c.AuthHandler(), c.AuthHandler())
	assertSingleton(t, "AuthMiddleware", c.AuthMiddleware(), c.AuthMiddleware())
	assertSingleton(t, "AccessMiddleware", c.AccessMiddleware(), c.AccessMiddleware())
}

func TestContainer_Router(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	assertSingleton(t, "Router", c.Router(), c.Router())
}

func TestContainer_RouterServesRequests(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())
	router := c.Router()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %q", rr.Body.String())
	}
}

func TestContainer_RouterAuthAndProjectFlow(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())
	router := c.Router()

	// Register
	regReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(testAuthJSON))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)

	if regRr.Code != http.StatusCreated {
		t.Fatalf("Register failed: status %d", regRr.Code)
	}

	// Login
	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(testAuthJSON))
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)

	if loginRr.Code != http.StatusOK {
		t.Fatalf("Login failed: status %d", loginRr.Code)
	}

	var loginResp map[string]string
	if err := json.NewDecoder(loginRr.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	token := loginResp["token"]

	// Create Project
	projID := uuid.New().String()
	projBody := fmt.Sprintf(`{"id":%q,"name":"DI Project"}`, projID)
	projReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString(projBody))
	projReq.Header.Set("Authorization", "Bearer "+token)
	projRr := httptest.NewRecorder()
	router.ServeHTTP(projRr, projReq)

	if projRr.Code != http.StatusCreated {
		t.Errorf("Create project failed: status %d, body %s", projRr.Code, projRr.Body.String())
	}
}
