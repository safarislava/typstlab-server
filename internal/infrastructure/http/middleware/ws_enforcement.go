package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	headerUpgrade             = "Upgrade"
	headerConnection          = "Connection"
	headerSecWebSocketKey     = "Sec-WebSocket-Key"
	headerSecWebSocketVersion = "Sec-WebSocket-Version"
	headerContentType         = "Content-Type"
	valueContentTypeJSON      = "application/json"
	valueWebSocket            = "websocket"
	valueUpgrade              = "Upgrade"
	valueUpgradeRequired      = "Upgrade Required"
	versionRFC6455            = "13"
	tokenUpgradeLower         = "upgrade"
	wsMandatoryMessage        = "WebSocket is mandatory for real-time collaboration. This device does not support RFC 6455."
)

type WSErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RequireWebSocket ensures incoming requests comply strictly with RFC 6455 WebSocket handshake requirements.
// If the client or device does not support WebSocket, the request is immediately rejected with 426 Upgrade Required.
func RequireWebSocket(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isWebSocketRequest(r) {
			w.Header().Set(headerUpgrade, valueWebSocket)
			w.Header().Set(headerConnection, valueUpgrade)
			w.Header().Set(headerSecWebSocketVersion, versionRFC6455)
			w.Header().Set(headerContentType, valueContentTypeJSON)
			w.WriteHeader(http.StatusUpgradeRequired)
			_ = json.NewEncoder(w).Encode(WSErrorResponse{
				Error:   valueUpgradeRequired,
				Message: wsMandatoryMessage,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isWebSocketRequest(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get(headerUpgrade), valueWebSocket) {
		return false
	}

	connHeader := r.Header.Get(headerConnection)
	if !headerContainsToken(connHeader, tokenUpgradeLower) {
		return false
	}

	if r.Header.Get(headerSecWebSocketKey) == "" {
		return false
	}

	if r.Header.Get(headerSecWebSocketVersion) != versionRFC6455 {
		return false
	}

	return true
}

func headerContainsToken(headerValue, targetToken string) bool {
	tokens := strings.SplitSeq(headerValue, ",")
	for token := range tokens {
		if strings.EqualFold(strings.TrimSpace(token), targetToken) {
			return true
		}
	}
	return false
}
