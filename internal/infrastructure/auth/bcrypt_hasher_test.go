package auth

import (
	"testing"
)

func TestBcryptHasher(t *testing.T) {
	t.Parallel()

	hasher := NewBcryptHasher(4)

	password := "my-secret-password"
	hashed, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Unexpected hash error: %v", err)
	}

	if hashed == "" {
		t.Fatal("Expected non-empty hash")
	}

	err = hasher.Compare(hashed, password)
	if err != nil {
		t.Errorf("Expected password to match: %v", err)
	}

	err = hasher.Compare(hashed, "wrong-password")
	if err == nil {
		t.Error("Expected mismatch error, got nil")
	}
}
