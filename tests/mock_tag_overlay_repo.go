package tests

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/model"
)

func CreateMockTagOverlayRepo() *MockTagOverlayRepo {
	return &MockTagOverlayRepo{Data: map[string]model.TagOverlay{}}
}

type MockTagOverlayRepo struct {
	model.TagOverlayRepository
	mu     sync.Mutex
	Data   map[string]model.TagOverlay
	Marked []string
	Err    error
}

func (m *MockTagOverlayRepo) Get(id string) (*model.TagOverlay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	o, ok := m.Data[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return &o, nil
}

func (m *MockTagOverlayRepo) GetAll() ([]model.TagOverlay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	return slices.Collect(maps.Values(m.Data)), nil
}

func (m *MockTagOverlayRepo) GetMany(ids ...string) (map[string]model.TagOverlay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	out := map[string]model.TagOverlay{}
	for _, id := range ids {
		if o, ok := m.Data[id]; ok {
			out[id] = o
		}
	}
	return out, nil
}

func (m *MockTagOverlayRepo) Put(o *model.TagOverlay) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	o.UpdatedAt = time.Now()
	m.Data[o.MediaFileID] = *o
	return nil
}

func (m *MockTagOverlayRepo) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	delete(m.Data, id)
	return nil
}

func (m *MockTagOverlayRepo) MarkForRescan(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.Marked = append(m.Marked, id)
	return nil
}
