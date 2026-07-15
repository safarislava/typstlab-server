package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()
	router := setupRouter()

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
	router := setupRouter()

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

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rr.Code)
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
	router := setupRouter()

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

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	t.Parallel()
	router := setupRouter()

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

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
