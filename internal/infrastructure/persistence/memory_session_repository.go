package persistence

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/safarislava/typstlab-server/internal/domain/session"
	domainToken "github.com/safarislava/typstlab-server/internal/domain/token"
)

type MemorySessionRepository struct {
	mu    sync.RWMutex
	store map[string]session.Session
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		store: make(map[string]session.Session),
	}
}

func (r *MemorySessionRepository) Save(_ context.Context, s session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[s.Token().Value()] = s
	return nil
}

func (r *MemorySessionRepository) FindByToken(_ context.Context, t domainToken.Token) (session.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.store[t.Value()]
	if !ok {
		return session.Session{}, session.ErrSessionNotFound
	}
	return s, nil
}

func (r *MemorySessionRepository) Delete(_ context.Context, t domainToken.Token) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, t.Value())
	return nil
}

func (r *MemorySessionRepository) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range r.store {
		if v.UserID() == userID {
			delete(r.store, k)
		}
	}
	return nil
}
