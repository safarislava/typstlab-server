package token

import (
	"errors"
	"testing"
)

func TestNewToken_Success(t *testing.T) {
	t.Parallel()

	val := "my-secret-token"
	tok, err := NewToken(val)
	if err != nil {
		t.Fatalf("Unexpected error creating token: %v", err)
	}

	if tok.Value() != val {
		t.Errorf("Expected token value %q, got %q", val, tok.Value())
	}
}

func TestNewToken_Error(t *testing.T) {
	t.Parallel()

	_, err := NewToken("")
	if !errors.Is(err, ErrInvalidTokenValue) {
		t.Errorf("Expected error %v, got %v", ErrInvalidTokenValue, err)
	}
}
