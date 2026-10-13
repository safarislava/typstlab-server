package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testSampleKey = "dGhlIHNhbXBsZSBub25jZQ=="
)

func TestRequireWebSocket_Success(t *testing.T) {
	t.Parallel()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireWebSocket(next)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
	req.Header.Set(headerUpgrade, valueWebSocket)
	req.Header.Set(headerConnection, valueUpgrade)
	req.Header.Set(headerSecWebSocketKey, testSampleKey)
	req.Header.Set(headerSecWebSocketVersion, versionRFC6455)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("Expected next handler to be called for valid WebSocket handshake")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
}

func TestRequireWebSocket_Success_ConnectionHeaderWithMultipleTokens(t *testing.T) {
	t.Parallel()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireWebSocket(next)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
	req.Header.Set(headerUpgrade, valueWebSocket)
	req.Header.Set(headerConnection, "keep-alive, Upgrade")
	req.Header.Set(headerSecWebSocketKey, testSampleKey)
	req.Header.Set(headerSecWebSocketVersion, versionRFC6455)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("Expected next handler to be called when Connection header contains 'upgrade'")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
}

type rejectionTestCase struct {
	name    string
	headers map[string]string
}

var rejectionTestCases = []rejectionTestCase{
	{
		name: "missing all websocket headers",
		headers: map[string]string{
			"User-Agent": "curl/8.7.1",
		},
	},
	{
		name: "missing upgrade header",
		headers: map[string]string{
			headerConnection:          valueUpgrade,
			headerSecWebSocketKey:     testSampleKey,
			headerSecWebSocketVersion: versionRFC6455,
		},
	},
	{
		name: "invalid upgrade header value",
		headers: map[string]string{
			headerUpgrade:             "h2c",
			headerConnection:          valueUpgrade,
			headerSecWebSocketKey:     testSampleKey,
			headerSecWebSocketVersion: versionRFC6455,
		},
	},
	{
		name: "missing connection header",
		headers: map[string]string{
			headerUpgrade:             valueWebSocket,
			headerSecWebSocketKey:     testSampleKey,
			headerSecWebSocketVersion: versionRFC6455,
		},
	},
	{
		name: "connection header without upgrade token",
		headers: map[string]string{
			headerUpgrade:             valueWebSocket,
			headerConnection:          "keep-alive",
			headerSecWebSocketKey:     testSampleKey,
			headerSecWebSocketVersion: versionRFC6455,
		},
	},
	{
		name: "missing sec-websocket-key",
		headers: map[string]string{
			headerUpgrade:             valueWebSocket,
			headerConnection:          valueUpgrade,
			headerSecWebSocketVersion: versionRFC6455,
		},
	},
	{
		name: "missing sec-websocket-version",
		headers: map[string]string{
			headerUpgrade:         valueWebSocket,
			headerConnection:      valueUpgrade,
			headerSecWebSocketKey: testSampleKey,
		},
	},
	{
		name: "unsupported sec-websocket-version",
		headers: map[string]string{
			headerUpgrade:             valueWebSocket,
			headerConnection:          valueUpgrade,
			headerSecWebSocketKey:     testSampleKey,
			headerSecWebSocketVersion: "8",
		},
	},
}

func TestRequireWebSocket_RejectionCases(t *testing.T) {
	t.Parallel()

	for _, tt := range rejectionTestCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertRejection(t, tt.headers)
		})
	}
}

func assertRejection(t *testing.T, headers map[string]string) {
	t.Helper()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireWebSocket(next)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("Next handler should NOT have been called for invalid WS handshake")
	}
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("Expected status %d (StatusUpgradeRequired), got %d", http.StatusUpgradeRequired, rec.Code)
	}

	verifyRejectionHeadersAndBody(t, rec)
}

func verifyRejectionHeadersAndBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get(headerUpgrade) != valueWebSocket {
		t.Errorf("Expected Upgrade header %q, got %q", valueWebSocket, rec.Header().Get(headerUpgrade))
	}
	if rec.Header().Get(headerConnection) != valueUpgrade {
		t.Errorf("Expected Connection header %q, got %q", valueUpgrade, rec.Header().Get(headerConnection))
	}
	if rec.Header().Get(headerSecWebSocketVersion) != versionRFC6455 {
		t.Errorf("Expected Sec-WebSocket-Version header %q, got %q", versionRFC6455, rec.Header().Get(headerSecWebSocketVersion))
	}

	var errResp WSErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode JSON error response: %v", err)
	}

	if errResp.Error != valueUpgradeRequired {
		t.Errorf("Expected Error %q, got %q", valueUpgradeRequired, errResp.Error)
	}
}
