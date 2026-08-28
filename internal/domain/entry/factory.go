package entry

import (
	"time"

	"github.com/google/uuid"

	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

// NewEntry creates and validates a new Entry entity.
func NewEntry(
	id uuid.UUID,
	name string,
	entryType domainFile.Type,
	isDeleted bool,
	updatedAt time.Time,
) (*Entry, error) {
	if id == uuid.Nil {
		return nil, ErrEmptyID
	}
	if name == "" {
		return nil, ErrEmptyName
	}
	return &Entry{
		id:        id,
		name:      name,
		entryType: entryType,
		isDeleted: isDeleted,
		updatedAt: updatedAt,
	}, nil
}
