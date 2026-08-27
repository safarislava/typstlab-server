package di

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
)

func TestContainer_LazyInitialization(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port:           "8080",
		DatabaseURL:    "postgres://user:pass@localhost:5432/db",
		JWTSecret:      "test-secret",
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	c := New(cfg)

	// Verify all instances are nil initially
	if c.projectRepo != nil || c.fileRepo != nil || c.userRepo != nil || c.sessionRepo != nil {
		t.Fatal("Repositories should not be initialized before being requested")
	}
	if c.hasher != nil || c.tokenService != nil || c.yjsMerger != nil {
		t.Fatal("Infrastructure components should not be initialized before being requested")
	}
	if c.projectService != nil || c.fileService != nil || c.syncService != nil || c.userService != nil || c.sessionService != nil || c.authService != nil {
		t.Fatal("Services should not be initialized before being requested")
	}
	if c.projectHandler != nil || c.fileHandler != nil || c.syncHandler != nil || c.userHandler != nil || c.authHandler != nil || c.authMiddleware != nil || c.accessMiddleware != nil {
		t.Fatal("Handlers and middlewares should not be initialized before being requested")
	}
	if c.router != nil {
		t.Fatal("Router should not be initialized before being requested")
	}

	// Verify Config getter
	if c.Config() != cfg {
		t.Errorf("Expected config %v, got %v", cfg, c.Config())
	}

	// Test Project components lazy loading & singleton caching
	pRepo1 := c.ProjectRepo()
	pRepo2 := c.ProjectRepo()
	if pRepo1 == nil || pRepo1 != pRepo2 {
		t.Error("ProjectRepo should be non-nil and return the same singleton instance")
	}

	pService1 := c.ProjectService()
	pService2 := c.ProjectService()
	if pService1 == nil || pService1 != pService2 {
		t.Error("ProjectService should be non-nil and return the same singleton instance")
	}

	pHandler1 := c.ProjectHandler()
	pHandler2 := c.ProjectHandler()
	if pHandler1 == nil || pHandler1 != pHandler2 {
		t.Error("ProjectHandler should be non-nil and return the same singleton instance")
	}

	// Test File & CRDT & Sync components
	fRepo1 := c.FileRepo()
	fRepo2 := c.FileRepo()
	if fRepo1 == nil || fRepo1 != fRepo2 {
		t.Error("FileRepo should be non-nil and return the same singleton instance")
	}

	merger1 := c.YjsMerger()
	merger2 := c.YjsMerger()
	if merger1 == nil || merger1 != merger2 {
		t.Error("YjsMerger should be non-nil and return the same singleton instance")
	}

	fService1 := c.FileService()
	fService2 := c.FileService()
	if fService1 == nil || fService1 != fService2 {
		t.Error("FileService should be non-nil and return the same singleton instance")
	}

	fHandler1 := c.FileHandler()
	fHandler2 := c.FileHandler()
	if fHandler1 == nil || fHandler1 != fHandler2 {
		t.Error("FileHandler should be non-nil and return the same singleton instance")
	}

	syncSvc1 := c.SyncService()
	syncSvc2 := c.SyncService()
	if syncSvc1 == nil || syncSvc1 != syncSvc2 {
		t.Error("SyncService should be non-nil and return the same singleton instance")
	}

	syncHandler1 := c.SyncHandler()
	syncHandler2 := c.SyncHandler()
	if syncHandler1 == nil || syncHandler1 != syncHandler2 {
		t.Error("SyncHandler should be non-nil and return the same singleton instance")
	}

	// Test User & Session & Auth components
	uRepo1 := c.UserRepo()
	uRepo2 := c.UserRepo()
	if uRepo1 == nil || uRepo1 != uRepo2 {
		t.Error("UserRepo should be non-nil and return the same singleton instance")
	}

	sRepo1 := c.SessionRepo()
	sRepo2 := c.SessionRepo()
	if sRepo1 == nil || sRepo1 != sRepo2 {
		t.Error("SessionRepo should be non-nil and return the same singleton instance")
	}

	hasher1 := c.Hasher()
	hasher2 := c.Hasher()
	if hasher1 == nil || hasher1 != hasher2 {
		t.Error("Hasher should be non-nil and return the same singleton instance")
	}

	tokenSvc1 := c.TokenService()
	tokenSvc2 := c.TokenService()
	if tokenSvc1 == nil || tokenSvc1 != tokenSvc2 {
		t.Error("TokenService should be non-nil and return the same singleton instance")
	}

	uService1 := c.UserService()
	uService2 := c.UserService()
	if uService1 == nil || uService1 != uService2 {
		t.Error("UserService should be non-nil and return the same singleton instance")
	}

	sService1 := c.SessionService()
	sService2 := c.SessionService()
	if sService1 == nil || sService1 != sService2 {
		t.Error("SessionService should be non-nil and return the same singleton instance")
	}

	authSvc1 := c.AuthService()
	authSvc2 := c.AuthService()
	if authSvc1 == nil || authSvc1 != authSvc2 {
		t.Error("AuthService should be non-nil and return the same singleton instance")
	}

	uHandler1 := c.UserHandler()
	uHandler2 := c.UserHandler()
	if uHandler1 == nil || uHandler1 != uHandler2 {
		t.Error("UserHandler should be non-nil and return the same singleton instance")
	}

	aHandler1 := c.AuthHandler()
	aHandler2 := c.AuthHandler()
	if aHandler1 == nil || aHandler1 != aHandler2 {
		t.Error("AuthHandler should be non-nil and return the same singleton instance")
	}

	authMid1 := c.AuthMiddleware()
	authMid2 := c.AuthMiddleware()
	if authMid1 == nil || authMid1 != authMid2 {
		t.Error("AuthMiddleware should be non-nil and return the same singleton instance")
	}

	accMid1 := c.AccessMiddleware()
	accMid2 := c.AccessMiddleware()
	if accMid1 == nil || accMid1 != accMid2 {
		t.Error("AccessMiddleware should be non-nil and return the same singleton instance")
	}

	// Router lazy load
	r1 := c.Router()
	r2 := c.Router()
	if r1 == nil || r1 != r2 {
		t.Error("Router should be non-nil and return the same singleton instance")
	}
}

func TestContainer_RouterServesRequests(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port:           "8080",
		DatabaseURL:    "postgres://user:pass@localhost:5432/db",
		JWTSecret:      "test-secret",
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	c := New(cfg)
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
