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
	testDatabaseURL   = "postgres://user:pass@localhost:5432/db"
	testJWTSecret     = "test-secret"
	testOrigin        = "http://localhost:3000"
	testAuthJSON      = `{"email":"di_user@example.com","password":"password123"}`
	mockAuth          = "test"
	cookieAccessToken = "access_token"
)

func newTestConfig() *config.Config {
	return &config.Config{
		Port:           "8080",
		DatabaseURL:    testDatabaseURL,
		JWTSecret:      testJWTSecret,
		AllowedOrigins: []string{testOrigin},
		S3: config.S3Config{
			Endpoint:  "localhost:9000",
			Bucket:    "test-bucket",
			AccessKey: mockAuth,
			SecretKey: mockAuth,
		},
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

	c := New(newTestConfig(), WithMemoryRepositories())

	assertNil(t, "projectRepo", c.projectRepo)
	assertNil(t, "inviteRepo", c.inviteRepo)
	assertNil(t, "fileRepo", c.fileRepo)
	assertNil(t, "userRepo", c.userRepo)
	assertNil(t, "sessionRepo", c.sessionRepo)

	assertNil(t, "hasher", c.hasher)
	assertNil(t, "tokenService", c.tokenService)
	assertNil(t, "yjsMerger", c.yjsMerger)

	assertNil(t, "projectService", c.projectService)
	assertNil(t, "inviteService", c.inviteService)
	assertNil(t, "typstFileService", c.typstFileService)
	assertNil(t, "binaryFileService", c.binaryFileService)
	assertNil(t, "fileService", c.fileService)
	assertNil(t, "syncService", c.syncService)
	assertNil(t, "userService", c.userService)
	assertNil(t, "sessionService", c.sessionService)
	assertNil(t, "authService", c.authService)

	assertNil(t, "projectHandler", c.projectHandler)
	assertNil(t, "fileHandler", c.fileHandler)
	assertNil(t, "wsHandler", c.wsHandler)
	assertNil(t, "userHandler", c.userHandler)
	assertNil(t, "authHandler", c.authHandler)
	assertNil(t, "authMiddleware", c.authMiddleware)
	assertNil(t, "accessMiddleware", c.accessMiddleware)

	assertNil(t, "router", c.router)
}

func TestContainer_ConfigAndLifecycle(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig()
	c := New(cfg, WithMemoryRepositories())

	if c.Config() != cfg {
		t.Errorf("Expected config %v, got %v", cfg, c.Config())
	}

	// Close on container with nil pool should not panic
	c.Close()
}

func TestContainer_Options(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig()
	c := New(cfg, WithPool(nil), WithS3Storage(nil), WithMemoryRepositories())

	if !c.useMemory {
		t.Error("Expected useMemory to be true")
	}
}

func TestContainer_Repositories(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())

	assertSingleton(t, "ProjectRepo", c.ProjectRepo(), c.ProjectRepo())
	assertSingleton(t, "InviteRepo", c.InviteRepo(), c.InviteRepo())
	assertSingleton(t, "FileRepo", c.FileRepo(), c.FileRepo())
	assertSingleton(t, "UserRepo", c.UserRepo(), c.UserRepo())
	assertSingleton(t, "SessionRepo", c.SessionRepo(), c.SessionRepo())
}

func TestContainer_Infrastructure(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())

	assertSingleton(t, "Hasher", c.Hasher(), c.Hasher())
	assertSingleton(t, "TokenService", c.TokenService(), c.TokenService())
	assertSingleton(t, "YjsMerger", c.YjsMerger(), c.YjsMerger())
	assertSingleton(t, "WSUpgrader", c.WSUpgrader(), c.WSUpgrader())
	assertSingleton(t, "Hub", c.Hub(), c.Hub())
}

func TestContainer_Services(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())

	assertSingleton(t, "ProjectService", c.ProjectService(), c.ProjectService())
	assertSingleton(t, "InviteService", c.InviteService(), c.InviteService())
	assertSingleton(t, "TypstFileService", c.TypstFileService(), c.TypstFileService())
	assertSingleton(t, "BinaryFileService", c.BinaryFileService(), c.BinaryFileService())
	assertSingleton(t, "FileService", c.FileService(), c.FileService())
	assertSingleton(t, "EntryService", c.EntryService(), c.EntryService())
	assertSingleton(t, "MetadataService", c.MetadataService(), c.MetadataService())
	assertSingleton(t, "SyncMetadataService", c.SyncMetadataService(), c.SyncMetadataService())
	assertSingleton(t, "SyncFileService", c.SyncFileService(), c.SyncFileService())
	assertSingleton(t, "SyncService", c.SyncService(), c.SyncService())
	assertSingleton(t, "CollaborationService", c.CollaborationService(), c.CollaborationService())
	assertSingleton(t, "UserService", c.UserService(), c.UserService())
	assertSingleton(t, "SessionService", c.SessionService(), c.SessionService())
	assertSingleton(t, "AuthService", c.AuthService(), c.AuthService())
}

func TestContainer_HandlersAndMiddlewares(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())

	assertSingleton(t, "ProjectHandler", c.ProjectHandler(), c.ProjectHandler())
	assertSingleton(t, "InviteHandler", c.InviteHandler(), c.InviteHandler())
	assertSingleton(t, "FileHandler", c.FileHandler(), c.FileHandler())
	assertSingleton(t, "WSHandler", c.WSHandler(), c.WSHandler())
	assertSingleton(t, "UserHandler", c.UserHandler(), c.UserHandler())
	assertSingleton(t, "AuthHandler", c.AuthHandler(), c.AuthHandler())
	assertSingleton(t, "AuthMiddleware", c.AuthMiddleware(), c.AuthMiddleware())
	assertSingleton(t, "AccessMiddleware", c.AccessMiddleware(), c.AccessMiddleware())
}

func TestContainer_Router(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())

	assertSingleton(t, "Router", c.Router(), c.Router())
}

func TestContainer_RouterServesRequests(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())
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

	c := New(newTestConfig(), WithMemoryRepositories())
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
	projReq.AddCookie(&http.Cookie{Name: cookieAccessToken, Value: token})
	projRr := httptest.NewRecorder()
	router.ServeHTTP(projRr, projReq)

	if projRr.Code != http.StatusCreated {
		t.Errorf("Create project failed: status %d, body %s", projRr.Code, projRr.Body.String())
	}
}

func TestContainer_RouterWSRejection(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())
	router := c.Router()

	projID := uuid.New().String()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws/projects/"+projID, http.NoBody)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUpgradeRequired {
		t.Errorf("Expected status %d (Upgrade Required), got %d", http.StatusUpgradeRequired, rr.Code)
	}
}

func registerAndLoginUser(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	regReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(body))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)
	if regRr.Code != http.StatusCreated {
		t.Fatalf("Register failed for %s: %d", email, regRr.Code)
	}

	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(body))
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)
	if loginRr.Code != http.StatusOK {
		t.Fatalf("Login failed for %s: %d", email, loginRr.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(loginRr.Body).Decode(&resp)
	return resp["token"]
}

func doAuthedRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: cookieAccessToken, Value: token})
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func requireStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int, msg string) {
	t.Helper()
	if rr.Code != expected {
		t.Fatalf("%s: expected %d, got %d (%s)", msg, expected, rr.Code, rr.Body.String())
	}
}

func TestContainer_ProjectMembersAndInvitesFlow(t *testing.T) {
	t.Parallel()

	c := New(newTestConfig(), WithMemoryRepositories())
	router := c.Router()

	token1 := registerAndLoginUser(t, router, "user1@example.com", "password123")
	_ = registerAndLoginUser(t, router, "user2@example.com", "password123")
	token3 := registerAndLoginUser(t, router, "user3@example.com", "password123")

	projID := uuid.New().String()
	projBody := fmt.Sprintf(`{"id":%q,"name":"Shared Project"}`, projID)
	createRr := doAuthedRequest(router, http.MethodPost, "/projects", token1, projBody)
	requireStatus(t, createRr, http.StatusCreated, "Create project")

	listMembersRr := doAuthedRequest(router, http.MethodGet, "/projects/"+projID+"/members", token1, "")
	requireStatus(t, listMembersRr, http.StatusOK, "List members")

	addMemberRr := doAuthedRequest(router, http.MethodPost, "/projects/"+projID+"/members", token1, `{"email":"user2@example.com"}`)
	requireStatus(t, addMemberRr, http.StatusCreated, "Add member")
	var addedMember map[string]string
	_ = json.NewDecoder(addMemberRr.Body).Decode(&addedMember)
	user2ID := addedMember["id"]

	createInvRr := doAuthedRequest(router, http.MethodPost, "/projects/"+projID+"/invites", token1, `{}`)
	requireStatus(t, createInvRr, http.StatusCreated, "Create invite")
	var invResp map[string]string
	_ = json.NewDecoder(createInvRr.Body).Decode(&invResp)
	inviteCode, inviteID := invResp["code"], invResp["id"]

	listInvRr := doAuthedRequest(router, http.MethodGet, "/projects/"+projID+"/invites", token1, "")
	requireStatus(t, listInvRr, http.StatusOK, "List invites")

	acceptRr := doAuthedRequest(router, http.MethodPost, "/invites/"+inviteCode+"/accept", token3, "")
	requireStatus(t, acceptRr, http.StatusOK, "Accept invite")

	getProjRr := doAuthedRequest(router, http.MethodGet, "/projects/"+projID, token3, "")
	requireStatus(t, getProjRr, http.StatusOK, "Get project")

	removeRr := doAuthedRequest(router, http.MethodDelete, "/projects/"+projID+"/members/"+user2ID, token1, "")
	requireStatus(t, removeRr, http.StatusNoContent, "Remove member")

	revokeRr := doAuthedRequest(router, http.MethodDelete, "/projects/"+projID+"/invites/"+inviteID, token1, "")
	requireStatus(t, revokeRr, http.StatusNoContent, "Revoke invite")
}
