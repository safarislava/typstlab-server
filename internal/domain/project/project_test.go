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
	name := "My Project"
	now := time.Now()

	p, err := NewProject(id, name, now)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if p == nil {
		t.Fatal("Expected project, got nil")
	}

	if p.ID() != id {
		t.Errorf("ID() = %v, want %v", p.ID(), id)
	}

	if p.Name() != name {
		t.Errorf("Name() = %q, want %q", p.Name(), name)
	}

	if p.UpdatedAt() != now {
		t.Errorf("UpdatedAt() = %v, want %v", p.UpdatedAt(), now)
	}
}

func TestNewProject_ValidationErrors(t *testing.T) {
	t.Parallel()

	const validName = "Valid Name"
	tests := []struct {
		name     string
		id       uuid.UUID
		projName string
		updated  time.Time
		wantErr  error
	}{
		{
			name:     "Empty ID",
			id:       uuid.Nil,
			projName: validName,
			updated:  time.Now(),
			wantErr:  ErrEmptyID,
		},
		{
			name:     "Empty Name",
			id:       uuid.New(),
			projName: "",
			updated:  time.Now(),
			wantErr:  ErrEmptyName,
		},
		{
			name:     "Zero UpdatedAt",
			id:       uuid.New(),
			projName: validName,
			updated:  time.Time{},
			wantErr:  ErrEmptyUpdatedAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := NewProject(tt.id, tt.projName, tt.updated)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewProject() error = %v, want = %v", err, tt.wantErr)
			}
			if p != nil {
				t.Error("NewProject() expected nil project on error, got non-nil")
			}
		})
	}
}
