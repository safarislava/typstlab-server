package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/project"
)

type ProjectRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.Project
}

func NewMemoryProjectRepository() *ProjectRepository {
	return &ProjectRepository{
		store: make(map[string]*domain.Project),
	}
}

func (r *ProjectRepository) Save(_ context.Context, p *domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[p.ID().String()] = p
	return nil
}

func (r *ProjectRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.store[id.String()]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}

	return p, nil
}
