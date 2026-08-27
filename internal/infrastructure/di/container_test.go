package di

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
)

const (
	testDatabaseURL = "postgres://user:pass@localhost:5432/db"
	testJWTSecret   = "test-secret"
	testOrigin      = "http://localhost:3000"
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

func TestContainer_InitialStateIsNil(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig())

	fields := []any{
		c.projectRepo, c.fileRepo, c.userRepo, c.sessionRepo,
		c.hasher, c.tokenService, c.yjsMerger,
		c.projectService, c.fileService, c.syncService,
		c.userService, c.sessionService, c.authService,
		c.projectHandler, c.fileHandler, c.syncHandler,
		c.userHandler, c.authHandler, c.authMiddleware, c.accessMiddleware,
		c.router,
	}

	for _, field := range fields {
		if field != nil {
			t.Fatal("Components should not be initialized before being requested")
		}
	}
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
