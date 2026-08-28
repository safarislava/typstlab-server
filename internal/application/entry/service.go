package entry

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
	domainFile "github.com/safarislava/typstlab-server/internal/domain/file"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// CreateEntry validates and instantiates a new domain Entry entity.
func (s *Service) CreateEntry(
	id uuid.UUID,
	name string,
	entryType domainFile.Type,
	isDeleted bool,
	updatedAt time.Time,
) (*domainEntry.Entry, error) {
	e, err := domainEntry.NewEntry(id, name, entryType, isDeleted, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}
	return e, nil
}

// Rename updates the name of an entry.
func (s *Service) Rename(e *domainEntry.Entry, newName string) error {
	if err := e.Rename(newName); err != nil {
		return fmt.Errorf("failed to rename entry: %w", err)
	}
	return nil
}

// Delete marks an entry as deleted.
func (s *Service) Delete(e *domainEntry.Entry) {
	e.MarkDeleted()
}
