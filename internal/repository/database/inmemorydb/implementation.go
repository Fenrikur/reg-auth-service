package inmemorydb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eurofurence/reg-auth-service/internal/entity"
	"github.com/eurofurence/reg-auth-service/internal/repository/database/dbrepo"
)

type InMemoryRepository struct {
	mu           sync.Mutex
	authRequests map[string]*entity.AuthRequest
}

func Create() dbrepo.Repository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Open() {
	r.authRequests = make(map[string]*entity.AuthRequest)
}

func (r *InMemoryRepository) Close() {
	r.authRequests = nil
}

func (r *InMemoryRepository) AddAuthRequest(ctx context.Context, ar *entity.AuthRequest) error {
	if r.authRequests == nil {
		return fmt.Errorf("cannot add auth request '%s' - repository closed", ar.State)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.authRequests[ar.State]; ok {
		return fmt.Errorf("cannot add auth request '%s' - already present", ar.State)
	} else {
		// copy the entity, so later modifications won't also modify it in the in-memory db
		copiedEntity := *ar
		r.authRequests[ar.State] = &copiedEntity
		return nil
	}
}

func (r *InMemoryRepository) GetAuthRequestByState(ctx context.Context, state string) (*entity.AuthRequest, error) {
	if r.authRequests == nil {
		return nil, fmt.Errorf("cannot add auth request '%s' - repository closed", state)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if ar, ok := r.authRequests[state]; ok {
		if ar.ExpiresAt.Before(time.Now()) {
			delete(r.authRequests, state)
			return nil, fmt.Errorf("cannot get auth request '%s' - already expired", state)
		} else {
			// copy the entity, so later modifications won't also modify it in the in-memory db
			copiedEntity := *ar
			return &copiedEntity, nil
		}
	} else {
		return nil, fmt.Errorf("cannot get auth request '%s' - not present", state)
	}
}

func (r *InMemoryRepository) DeleteAuthRequestByState(ctx context.Context, state string) error {
	if r.authRequests == nil {
		return fmt.Errorf("cannot add auth request '%s' - repository closed", state)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.authRequests[state]; ok {
		delete(r.authRequests, state)
		return nil
	} else {
		return fmt.Errorf("cannot delete auth request '%s' - not present", state)
	}
}

func (r *InMemoryRepository) PruneAuthRequests(ctx context.Context) (uint, error) {
	if r.authRequests == nil {
		return 0, fmt.Errorf("cannot prune auth requests - repository closed")
	}

	pruneCount := uint(0)

	r.mu.Lock()
	for state, ar := range r.authRequests {
		if ar.ExpiresAt.Before(time.Now()) {
			delete(r.authRequests, state)
			pruneCount++
		}
	}
	r.mu.Unlock()

	return pruneCount, nil
}