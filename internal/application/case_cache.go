package application

import (
	"dialect-release/internal/domain"
	"sync"
)

type caseSnapshotCache struct {
	mu     sync.RWMutex
	values map[string]*domain.ReleaseCase
}

func newCaseSnapshotCache() *caseSnapshotCache {
	return &caseSnapshotCache{values: make(map[string]*domain.ReleaseCase)}
}

func (c *caseSnapshotCache) get(id string) (*domain.ReleaseCase, bool, error) {
	c.mu.RLock()
	value, ok := c.values[id]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	cloned, err := domain.CloneCase(value)
	return cloned, true, err
}

func (c *caseSnapshotCache) put(id string, value *domain.ReleaseCase) error {
	cloned, err := domain.CloneCase(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.values[id] = cloned
	c.mu.Unlock()
	return nil
}

func (c *caseSnapshotCache) delete(id string) {
	c.mu.Lock()
	delete(c.values, id)
	c.mu.Unlock()
}
