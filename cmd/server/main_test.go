package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	yjs "github.com/reearth/ygo/crdt"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
	"github.com/safarislava/typstlab-server/internal/infrastructure/di"
)

const typstlabLink = "https://typstlab.safarislava.tech"

func setupTestRouter() *chi.Mux {
	cfg := config.Load("../../configs/config.json")
	return di.New(cfg).Router()
}

func registerAndLogin(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()

	regBody := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	regReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(regBody))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)
	if regRr.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: status %d, body %s", regRr.Code, regRr.Body.String())
	}

	loginBody := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	loginReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(loginBody))
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)
	if loginRr.Code != http.StatusOK {
		t.Fatalf("Failed to login user: status %d, body %s", loginRr.Code, loginRr.Body.String())
	}

	var loginResp map[string]string
	if err := json.NewDecoder(loginRr.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	return loginResp["token"]
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	expectedBody := "OK"
	if rr.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rr.Body.String())
	}
}

func TestCreateProject(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	token := registerAndLogin(t, router, "test@example.com", "secretpassword")

	projectID := uuid.New().String()
	body := fmt.Sprintf(`{"id":%q,"name":"My Test Project"}`, projectID)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/projects",
		bytes.NewBufferString(body),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d, body %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["name"] != "My Test Project" {
		t.Errorf("Expected project name 'My Test Project', got %q", resp["name"])
	}
	if resp["id"] != projectID {
		t.Errorf("Expected project ID %q, got %q", projectID, resp["id"])
	}
	if resp["updated_at"] == "" {
		t.Error("Expected updated_at in response, got empty string")
	}
}

func TestCreateProject_InvalidJSON(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	reqBody := `{invalid-json`
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/projects",
		bytes.NewBufferString(reqBody),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	token := registerAndLogin(t, router, "test_invalid_json@example.com", "password")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	token := registerAndLogin(t, router, "test2@example.com", "secretpassword")

	reqBody := fmt.Sprintf(`{"id":%q,"name":""}`, uuid.New().String())
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/projects",
		bytes.NewBufferString(reqBody),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestApplyFileChanges(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	token := registerAndLogin(t, router, "test_changes@example.com", "secretpassword")

	// 1. Create Project
	projectID := uuid.New().String()
	createProjectBody := fmt.Sprintf(`{"id":%q,"name":"My Test Project"}`, projectID)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString(createProjectBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d, body %s", http.StatusCreated, rr.Code, rr.Body.String())
	}
	var projectResp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&projectResp)

	// 2. Upload Typst File
	fileID := uuid.New().String()
	uploadBody := fmt.Sprintf(`{"id":%q,"name":"test.typxml","content":""}`, fileID)
	uploadReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/projects/"+projectID+"/files", bytes.NewBufferString(uploadBody))
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadRr := httptest.NewRecorder()
	router.ServeHTTP(uploadRr, uploadReq)
	if uploadRr.Code != http.StatusCreated {
		t.Fatalf("Expected upload status code %d, got %d, body %s", http.StatusCreated, uploadRr.Code, uploadRr.Body.String())
	}

	// 3. Apply File Changes
	doc := yjs.New()
	delta := doc.EncodeStateAsUpdate()
	deltaB64 := base64.StdEncoding.EncodeToString(delta)

	changesBody := fmt.Sprintf(`{"delta":%q}`, deltaB64)
	changesReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/files/typst/"+fileID+"/changes", bytes.NewBufferString(changesBody))
	changesReq.Header.Set("Authorization", "Bearer "+token)
	changesRr := httptest.NewRecorder()
	router.ServeHTTP(changesRr, changesReq)

	if changesRr.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d, body: %s", http.StatusOK, changesRr.Code, changesRr.Body.String())
	}
}

func TestCORS(t *testing.T) {
	t.Parallel()
	router := setupTestRouter()

	// 1. Preflight request (OPTIONS)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodOptions, "/health", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create preflight request: %v", err)
	}
	req.Header.Set("Origin", typstlabLink)
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Errorf("Expected preflight status 200 or 204, got %d", rr.Code)
	}

	allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != typstlabLink && allowOrigin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to match origin, got %q", allowOrigin)
	}

	// 2. Actual GET request with Origin
	getReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create GET request: %v", err)
	}
	getReq.Header.Set("Origin", typstlabLink)

	getRr := httptest.NewRecorder()
	router.ServeHTTP(getRr, getReq)

	if getRr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", getRr.Code)
	}

	getAllowOrigin := getRr.Header().Get("Access-Control-Allow-Origin")
	if getAllowOrigin != typstlabLink && getAllowOrigin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin on GET request, got %q", getAllowOrigin)
	}
}
