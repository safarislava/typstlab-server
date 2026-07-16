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

	// 1. Register User
	regBody := `{"email":"test@example.com","password":"secretpassword"}`
	regReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(regBody))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)
	if regRr.Code != http.StatusCreated {
		t.Fatalf("Failed to register user: status %d, body %s", regRr.Code, regRr.Body.String())
	}

	// 2. Login User
	loginBody := `{"email":"test@example.com","password":"secretpassword"}`
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
	token := loginResp["token"]

	// 3. Create Project with Token
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

	// Since it has invalid JSON, auth check is not reached because router middleware runs first.
	// But wait, the route is protected by RequireAuthenticated middleware.
	// So if no header is present, it returns 401 Unauthorized instead of 400 Bad Request!
	// Let's verify: indeed, RequireAuthenticated runs first in the middleware chain.
	// Let's pass a valid token or check for 401. Let's make this test pass by passing a token, or expect 401.
	// To test invalid JSON on the handler, we must first authenticate.
	// Let's register, login, and pass token. That is much better!
	regBody := `{"email":"test_invalid_json@example.com","password":"password"}`
	regReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(regBody))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)

	loginBody := `{"email":"test_invalid_json@example.com","password":"password"}`
	loginReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(loginBody))
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)

	var loginResp map[string]string
	_ = json.NewDecoder(loginRr.Body).Decode(&loginResp)
	token := loginResp["token"]

	req.Header.Set("Authorization", "Bearer "+token)

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateProject_ValidationError(t *testing.T) {
	t.Parallel()
	router := setupRouter()

	// 1. Register User
	regBody := `{"email":"test2@example.com","password":"secretpassword"}`
	regReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register", bytes.NewBufferString(regBody))
	regRr := httptest.NewRecorder()
	router.ServeHTTP(regRr, regReq)

	// 2. Login User
	loginBody := `{"email":"test2@example.com","password":"secretpassword"}`
	loginReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/login", bytes.NewBufferString(loginBody))
	loginRr := httptest.NewRecorder()
	router.ServeHTTP(loginRr, loginReq)

	var loginResp map[string]string
	_ = json.NewDecoder(loginRr.Body).Decode(&loginResp)
	token := loginResp["token"]

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
