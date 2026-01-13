package query

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SavedQueryRepo stores saved queries in memory.
type SavedQueryRepo struct {
	mu     sync.Mutex
	items  map[string]SavedQuery
	order  []string
}

func NewSavedQueryRepo() *SavedQueryRepo {
	return &SavedQueryRepo{items: make(map[string]SavedQuery)}
}

func (r *SavedQueryRepo) List(ctx context.Context) ([]SavedQuery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]SavedQuery, 0, len(r.order))
	for _, id := range r.order {
		result = append(result, r.items[id])
	}
	return result, nil
}

func (r *SavedQueryRepo) Create(ctx context.Context, name, description string, req QueryRequest) (SavedQuery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := fmt.Sprintf("q-%d", time.Now().UnixNano())
	sq := SavedQuery{
		ID:          id,
		Name:        name,
		Description: description,
		Request:     req,
		CreatedAt:   time.Now(),
	}
	r.items[id] = sq
	r.order = append(r.order, id)
	return sq, nil
}

func (r *SavedQueryRepo) Get(ctx context.Context, id string) (SavedQuery, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sq, ok := r.items[id]
	return sq, ok
}

func (r *SavedQueryRepo) Delete(ctx context.Context, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, id)
}
