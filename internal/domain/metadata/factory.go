package metadata

import (
	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
)

// NewMetadata creates and validates a new Metadata aggregate.
func NewMetadata(projectID uuid.UUID, entries []*domainEntry.Entry) (*Metadata, error) {
	if projectID == uuid.Nil {
		return nil, ErrEmptyProjectID
	}
	m := &Metadata{
		projectID: projectID,
		entries:   make(map[uuid.UUID]*domainEntry.Entry, len(entries)),
	}
	for _, e := range entries {
		if e != nil {
			m.entries[e.ID()] = e
		}
	}
	return m, nil
}
