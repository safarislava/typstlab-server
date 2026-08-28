package metadata

import (
	"maps"

	"github.com/google/uuid"

	domainEntry "github.com/safarislava/typstlab-server/internal/domain/entry"
)

// Metadata encapsulates the collection of file entries for a project.
type Metadata struct {
	projectID uuid.UUID
	entries   map[uuid.UUID]*domainEntry.Entry
}

func (m *Metadata) ProjectID() uuid.UUID {
	return m.projectID
}

func (m *Metadata) Entries() map[uuid.UUID]*domainEntry.Entry {
	res := make(map[uuid.UUID]*domainEntry.Entry, len(m.entries))
	maps.Copy(res, m.entries)
	return res
}

func (m *Metadata) ActiveEntries() []*domainEntry.Entry {
	var res []*domainEntry.Entry
	for _, e := range m.entries {
		if !e.IsDeleted() {
			res = append(res, e)
		}
	}
	return res
}

func (m *Metadata) Get(id uuid.UUID) (*domainEntry.Entry, bool) {
	e, ok := m.entries[id]
	return e, ok
}

func (m *Metadata) Set(e *domainEntry.Entry) {
	if e != nil {
		m.entries[e.ID()] = e
	}
}
