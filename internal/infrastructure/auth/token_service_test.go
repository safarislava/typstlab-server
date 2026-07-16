package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/user"
)

const (
	testSecretKey = "my-secret-key-for-testing-purposes"
)

func TestJWTTokenService_Success(t *testing.T) {
	t.Parallel()

	tokenService := NewJWTTokenService(testSecretKey, 1*time.Hour)

	userID := uuid.New()
	role := user.RoleUser

	token, err := tokenService.Generate(userID, role)
	if err != nil {
		t.Fatalf("Unexpected error generating token: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	parsedUserID, parsedRole, err := tokenService.Validate(token)
	if err != nil {
		t.Fatalf("Unexpected error validating token: %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("Validate() userID = %v, want %v", parsedUserID, userID)
	}

	if parsedRole != role {
		t.Errorf("Validate() role = %v, want %v", parsedRole, role)
	}
}

func TestJWTTokenService_Expired(t *testing.T) {
	t.Parallel()

	tokenService := NewJWTTokenService(testSecretKey, -10*time.Minute)

	userID := uuid.New()
	role := user.RoleAdmin

	token, err := tokenService.Generate(userID, role)
	if err != nil {
		t.Fatalf("Unexpected error generating token: %v", err)
	}

	_, _, err = tokenService.Validate(token)
	if err == nil {
		t.Fatal("Expected error validating expired token, got nil")
	}
}

func TestJWTTokenService_InvalidSignature(t *testing.T) {
	t.Parallel()

	tokenService1 := NewJWTTokenService("secret-key-1", 1*time.Hour)
	tokenService2 := NewJWTTokenService("secret-key-2", 1*time.Hour)

	userID := uuid.New()
	role := user.RoleUser

	token, err := tokenService1.Generate(userID, role)
	if err != nil {
		t.Fatalf("Unexpected error generating token: %v", err)
	}

	_, _, err = tokenService2.Validate(token)
	if err == nil {
		t.Fatal("Expected validation to fail due to key mismatch, got nil")
	}
}
