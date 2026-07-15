package persistence

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type MemoryProjectRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.Project
}

func NewMemoryProjectRepository() *MemoryProjectRepository {
	return &MemoryProjectRepository{
		store: make(map[string]*domain.Project),
	}
}

func (r *MemoryProjectRepository) Save(_ context.Context, p *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[p.ID().String()] = p
	return nil
}

func (r *MemoryProjectRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.store[id.String()]
	if !ok {
		return nil, errors.New("project not found")
	}

	return p, nil
}
