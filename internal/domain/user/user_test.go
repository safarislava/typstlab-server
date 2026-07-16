package user

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewUser_Success(t *testing.T) {
	t.Parallel()

	const email = "test@example.com"
	id := uuid.New()
	hash := "somehash"
	role := RoleUser

	u, err := NewUser(id, email, hash, role)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if u == nil {
		t.Fatal("Expected non-nil user")
	}

	if u.ID() != id {
		t.Errorf("ID() = %v, want %v", u.ID(), id)
	}
	if u.Email() != email {
		t.Errorf("Email() = %q, want %q", u.Email(), email)
	}
	if u.PasswordHash() != hash {
		t.Errorf("PasswordHash() = %q, want %q", u.PasswordHash(), hash)
	}
	if u.Role() != role {
		t.Errorf("Role() = %v, want %v", u.Role(), role)
	}
}

func TestNewUser_ValidationErrors(t *testing.T) {
	t.Parallel()

	const (
		testEmail = "a@b.com"
		testHash  = "hash"
	)

	tests := []struct {
		name    string
		id      uuid.UUID
		email   string
		hash    string
		role    Role
		wantErr error
	}{
		{name: "Empty ID", id: uuid.Nil, email: testEmail, hash: testHash, role: RoleUser, wantErr: ErrEmptyID},
		{name: "Invalid Email", id: uuid.New(), email: "invalid", hash: testHash, role: RoleUser, wantErr: ErrInvalidEmail},
		{name: "Empty Password Hash", id: uuid.New(), email: testEmail, hash: "", role: RoleUser, wantErr: ErrEmptyPasswordHash},
		{name: "Invalid Role", id: uuid.New(), email: testEmail, hash: testHash, role: Role("guest"), wantErr: ErrInvalidRole},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := NewUser(tt.id, tt.email, tt.hash, tt.role)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewUser() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if u != nil {
				t.Error("Expected nil user on error, got non-nil")
			}
		})
	}
}
