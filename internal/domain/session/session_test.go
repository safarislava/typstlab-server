package session

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/token"
)

func TestNewSession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	expiresAt := time.Now().Add(1 * time.Hour)
	rtVal, _ := token.NewToken("my-refresh-token")
	rt, err := NewSession(rtVal, userID, expiresAt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if rt.Token() != rtVal {
		t.Errorf("Expected token %v, got %v", rtVal, rt.Token())
	}
	if rt.Token().Value() != "my-refresh-token" {
		t.Errorf("Expected token value 'my-refresh-token', got %q", rt.Token().Value())
	}
	if rt.UserID() != userID {
		t.Errorf("Expected userID %s, got %s", userID, rt.UserID())
	}
	if rt.ExpiresAt() != expiresAt {
		t.Errorf("Expected expiresAt %v, got %v", expiresAt, rt.ExpiresAt())
	}
	if rt.IsExpired() {
		t.Error("Expected session not to be expired")
	}
}

func TestNewSession_Errors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Case 1: Empty token string / zero value token
	_, err := NewSession(token.Token{}, userID, time.Now().Add(1*time.Hour))
	if !errors.Is(err, token.ErrInvalidTokenValue) {
		t.Errorf("Expected ErrInvalidTokenValue, got %v", err)
	}

	// Case 2: Nil user ID
	rtVal, _ := token.NewToken("token")
	_, err = NewSession(rtVal, uuid.Nil, time.Now().Add(1*time.Hour))
	if !errors.Is(err, ErrInvalidUserID) {
		t.Errorf("Expected ErrInvalidUserID, got %v", err)
	}

	// Case 3: Expired time
	_, err = NewSession(rtVal, userID, time.Now().Add(-1*time.Hour))
	if !errors.Is(err, ErrExpiredSession) {
		t.Errorf("Expected ErrExpiredSession, got %v", err)
	}
}
