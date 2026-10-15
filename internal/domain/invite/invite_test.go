package invite

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInvite(t *testing.T) {
	t.Parallel()

	inviteID := uuid.New()
	projectID := uuid.New()
	createdBy := uuid.New()
	code := "test-code-123"
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	inv, err := NewInvite(inviteID, projectID, createdBy, code, expiresAt, now)
	if err != nil {
		t.Fatalf("Failed to create invite: %v", err)
	}

	if inv.ID() != inviteID || inv.ProjectID() != projectID || inv.CreatedBy() != createdBy || inv.Code() != code {
		t.Error("Invite fields mismatch")
	}
	if inv.ExpiresAt() != expiresAt || inv.CreatedAt() != now {
		t.Error("Invite timestamps mismatch")
	}

	if inv.IsExpired(now) {
		t.Error("Invite should not be expired at creation time")
	}
	if !inv.IsExpired(now.Add(25 * time.Hour)) {
		t.Error("Invite should be expired after expiration time")
	}

	// Validation tests
	tests := []struct {
		name      string
		id        uuid.UUID
		projectID uuid.UUID
		createdBy uuid.UUID
		code      string
		expiresAt time.Time
		wantErr   error
	}{
		{name: "Empty ID", id: uuid.Nil, projectID: projectID, createdBy: createdBy, code: code, expiresAt: expiresAt, wantErr: ErrEmptyInviteID},
		{name: "Empty Project ID", id: inviteID, projectID: uuid.Nil, createdBy: createdBy, code: code, expiresAt: expiresAt, wantErr: ErrEmptyProjectID},
		{name: "Empty CreatedBy", id: inviteID, projectID: projectID, createdBy: uuid.Nil, code: code, expiresAt: expiresAt, wantErr: ErrEmptyCreatedBy},
		{name: "Empty Code", id: inviteID, projectID: projectID, createdBy: createdBy, code: "", expiresAt: expiresAt, wantErr: ErrEmptyInviteCode},
		{name: "Empty ExpiresAt", id: inviteID, projectID: projectID, createdBy: createdBy, code: code, expiresAt: time.Time{}, wantErr: ErrEmptyExpiresAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewInvite(tt.id, tt.projectID, tt.createdBy, tt.code, tt.expiresAt, now)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewInvite() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}
