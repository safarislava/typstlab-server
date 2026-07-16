package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/safarislava/typstlab-server/internal/infrastructure/config"
)

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

	reqBody := `{"name":"My Test Project"}`
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
