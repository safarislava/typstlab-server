package entry

import (
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

// Entry represents a file or directory node in a project metadata tree.
type Entry struct {
	id        uuid.UUID
	name      string
	entryType domainFile.Type
	isDeleted bool
	updatedAt time.Time
}

func (e *Entry) ID() uuid.UUID {
	return e.id
}

func (e *Entry) Name() string {
	return e.name
}

func (e *Entry) Type() domainFile.Type {
	return e.entryType
}

func (e *Entry) IsDeleted() bool {
	return e.isDeleted
}

func (e *Entry) UpdatedAt() time.Time {
	return e.updatedAt
}

func (e *Entry) Rename(newName string) error {
	if newName == "" {
		return ErrEmptyName
	}
	e.name = newName
	e.updatedAt = time.Now()
	return nil
}

func (e *Entry) MarkDeleted() {
	e.isDeleted = true
	e.updatedAt = time.Now()
}
