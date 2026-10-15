package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/invite"
)

type InviteRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.Invite
}

func NewMemoryInviteRepository() *InviteRepository {
	return &InviteRepository{
		store: make(map[string]*domain.Invite),
	}
}

func (r *InviteRepository) Save(_ context.Context, inv *domain.Invite) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[inv.ID().String()] = inv
	return nil
}

func (r *InviteRepository) FindByCode(_ context.Context, code string) (*domain.Invite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, inv := range r.store {
		if inv.Code() == code {
			return inv, nil
		}
	}

	return nil, domain.ErrInviteNotFound
}

func (r *InviteRepository) FindByProjectID(_ context.Context, projectID uuid.UUID) ([]*domain.Invite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Invite
	for _, inv := range r.store {
		if inv.ProjectID() == projectID {
			result = append(result, inv)
		}
	}

	return result, nil
}

func (r *InviteRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Invite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inv, ok := r.store[id.String()]
	if !ok {
		return nil, domain.ErrInviteNotFound
	}

	return inv, nil
}

func (r *InviteRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.store, id.String())
	return nil
}
