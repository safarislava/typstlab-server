package project

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProject_Success(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	userIDs := []uuid.UUID{userID1, userID2}
	name := "My Project"
	now := time.Now()

	p, err := NewProject(id, userIDs, name, now)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if p.ID() != id || p.Name() != name || p.UpdatedAt() != now {
		t.Error("Project properties mismatch")
	}

	resUserIDs := p.UserIDs()
	if len(resUserIDs) != 2 || resUserIDs[0] != userID1 || resUserIDs[1] != userID2 {
		t.Errorf("UserIDs() = %v, want %v", resUserIDs, userIDs)
	}
}

func TestProject_Collaborators(t *testing.T) {
	t.Parallel()

	userID1 := uuid.New()
	userID2 := uuid.New()
	p, _ := NewProject(uuid.New(), []uuid.UUID{userID1}, "Collab Proj", time.Now())

	if !p.HasUser(userID1) {
		t.Error("Expected project to have initial owner")
	}
	if p.HasUser(userID2) {
		t.Error("Expected project to not have non-associated user")
	}

	if err := p.AddUser(userID2); err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}
	if !p.HasUser(userID2) {
		t.Error("Expected project to have user after AddUser")
	}

	if err := p.AddUser(uuid.Nil); err == nil {
		t.Error("Expected error when adding Nil UUID, got nil")
	}
}

func TestNewProject_ValidationErrors(t *testing.T) {
	t.Parallel()

	const validName = "Valid Name"
	validUserID := uuid.New()
	tests := []struct {
		name     string
		id       uuid.UUID
		userIDs  []uuid.UUID
		projName string
		updated  time.Time
		wantErr  error
	}{
		{name: "Empty ID", id: uuid.Nil, userIDs: []uuid.UUID{validUserID}, projName: validName, updated: time.Now(), wantErr: ErrEmptyID},
		{name: "Empty User IDs", id: uuid.New(), userIDs: []uuid.UUID{}, projName: validName, updated: time.Now(), wantErr: ErrNoUsers},
		{name: "Nil User ID", id: uuid.New(), userIDs: []uuid.UUID{uuid.Nil}, projName: validName, updated: time.Now(), wantErr: ErrEmptyUserID},
		{name: "Empty Name", id: uuid.New(), userIDs: []uuid.UUID{validUserID}, projName: "", updated: time.Now(), wantErr: ErrEmptyName},
		{name: "Zero UpdatedAt", id: uuid.New(), userIDs: []uuid.UUID{validUserID}, projName: validName, updated: time.Time{}, wantErr: ErrEmptyUpdatedAt},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewProject(tt.id, tt.userIDs, tt.projName, tt.updated)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewProject() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}
