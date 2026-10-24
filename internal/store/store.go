// Package store persists posture reviews so they survive pod restarts and
// multiple replicas. DynamoDB is the primary backend (serverless, cheap);
// an in-memory store is the fallback so the product always works.
package store

import (
	"context"
	"sort"
	"sync"

	"github.com/sriramg-aivar/nexogent/internal/types"
)

// Store persists and retrieves reviews.
type Store interface {
	Put(ctx context.Context, r *types.PostureReview) error
	Get(ctx context.Context, id string) (*types.PostureReview, error)
	Latest(ctx context.Context) (*types.PostureReview, error)
	List(ctx context.Context, limit int) ([]*types.PostureReview, error)
	Kind() string
}

// Memory is an in-memory Store (used when DynamoDB is not configured).
type Memory struct {
	mu      sync.RWMutex
	byID    map[string]*types.PostureReview
	order   []string // review ids, newest last
}

func NewMemory() *Memory {
	return &Memory{byID: map[string]*types.PostureReview{}}
}

func (m *Memory) Kind() string { return "memory" }

func (m *Memory) Put(_ context.Context, r *types.PostureReview) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[r.ReviewID]; !ok {
		m.order = append(m.order, r.ReviewID)
	}
	m.byID[r.ReviewID] = r
	return nil
}

func (m *Memory) Get(_ context.Context, id string) (*types.PostureReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.byID[id], nil
}

func (m *Memory) Latest(_ context.Context) (*types.PostureReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.order) == 0 {
		return nil, nil
	}
	return m.byID[m.order[len(m.order)-1]], nil
}

func (m *Memory) List(_ context.Context, limit int) ([]*types.PostureReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*types.PostureReview
	for i := len(m.order) - 1; i >= 0; i-- {
		out = append(out, m.byID[m.order[i]])
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
