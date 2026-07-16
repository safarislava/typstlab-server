package persistence

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	domain "github.com/safarislava/typstlab-server/internal/domain/user"
)

type MemoryUserRepository struct {
	mu    sync.RWMutex
	store map[string]*domain.User
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		store: make(map[string]*domain.User),
	}
}

func (r *MemoryUserRepository) Save(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[u.ID().String()] = u
	return nil
}

func (r *MemoryUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.store {
		if u.Email() == email {
			return u, nil
		}
	}

	return nil, errors.New("user not found")
}

func (r *MemoryUserRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.store[id.String()]
	if !ok {
		return nil, errors.New("user not found")
	}

	return u, nil
}
