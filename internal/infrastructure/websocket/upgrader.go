package websocket

import (
	"fmt"
	"net/http"

	"github.com/coder/websocket"
)

// Upgrader handles upgrading standard HTTP connections to WebSocket RFC 6455 connections.
type Upgrader struct {
	allowedOrigins []string
}

// NewUpgrader creates a new Upgrader with the specified allowed CORS origins.
func NewUpgrader(allowedOrigins []string) *Upgrader {
	return &Upgrader{
		allowedOrigins: allowedOrigins,
	}
}

// Upgrade upgrades the HTTP connection to a WebSocket connection.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	opts := &websocket.AcceptOptions{
		OriginPatterns: u.allowedOrigins,
	}
	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		return nil, fmt.Errorf("accept websocket connection: %w", err)
	}
	return conn, nil
}
