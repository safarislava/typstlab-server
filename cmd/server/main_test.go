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
)

const _reqBody = `{"name":"My Test Project"}`

func setupTestRouter() *chi.Mux {
	cfg := config.Load("../../configs/config.json")
	return setupRouter(cfg)
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

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/projects",
		bytes.NewBufferString(_reqBody),
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
	if resp["id"] == "" {
		t.Error("Expected project ID to be generated, got empty string")
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

	reqBody := `{"name":""}`
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
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/projects", bytes.NewBufferString(_reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d, body %s", http.StatusCreated, rr.Code, rr.Body.String())
	}
	var projectResp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&projectResp)
	projectID := projectResp["id"]

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
