package framework

import "sync"

// MemoryStore provides cross-walk, identity-scoped key-value persistence.
// Walker identity (walkerID) is the scoping dimension: each walker has
// its own namespace. Values persist across multiple graph walks when
// the same MemoryStore instance is reused.
type MemoryStore interface {
	Get(walkerID, key string) (any, bool)
	Set(walkerID, key string, value any)
	Keys(walkerID string) []string
}

// InMemoryStore is a thread-safe in-process MemoryStore.
type InMemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

// NewInMemoryStore creates a ready-to-use InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]map[string]any),
	}
}

func (s *InMemoryStore) Get(walkerID, key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.data[walkerID]
	if !ok {
		return nil, false
	}
	v, ok := ns[key]
	return v, ok
}

func (s *InMemoryStore) Set(walkerID, key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns, ok := s.data[walkerID]
	if !ok {
		ns = make(map[string]any)
		s.data[walkerID] = ns
	}
	ns[key] = value
}

func (s *InMemoryStore) Keys(walkerID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.data[walkerID]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(ns))
	for k := range ns {
		keys = append(keys, k)
	}
	return keys
}
