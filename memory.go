package framework

import (
	"strings"
	"sync"
	"time"
)

// MemoryStore provides cross-walk, identity-scoped key-value persistence.
// Walker identity (walkerID) is the scoping dimension: each walker has
// its own namespace. Values persist across multiple graph walks when
// the same MemoryStore instance is reused.
//
// The namespace-aware methods (GetNS, SetNS, KeysNS, Search) add a second
// scoping dimension. The original Get/Set/Keys use a default namespace ("").
type MemoryStore interface {
	Get(walkerID, key string) (any, bool)
	Set(walkerID, key string, value any)
	Keys(walkerID string) []string

	GetNS(namespace, walkerID, key string) (any, bool)
	SetNS(namespace, walkerID, key string, value any)
	KeysNS(namespace, walkerID string) []string
	Search(namespace, query string) []MemoryItem
}

// Conventional namespace constants for the three memory types.
const (
	NamespaceSemantic   = "semantic"
	NamespaceEpisodic   = "episodic"
	NamespaceProcedural = "procedural"
)

// MemoryItem represents a stored memory entry with metadata.
type MemoryItem struct {
	Namespace string    `json:"namespace"`
	WalkerID  string    `json:"walker_id"`
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// InMemoryStore is a thread-safe in-process MemoryStore with namespace support.
type InMemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string]map[string]MemoryItem // namespace -> walkerID -> key -> item
}

// NewInMemoryStore creates a ready-to-use InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]map[string]map[string]MemoryItem),
	}
}

// --- Backward-compatible methods (default namespace "")  ---

func (s *InMemoryStore) Get(walkerID, key string) (any, bool) {
	return s.GetNS("", walkerID, key)
}

func (s *InMemoryStore) Set(walkerID, key string, value any) {
	s.SetNS("", walkerID, key, value)
}

func (s *InMemoryStore) Keys(walkerID string) []string {
	return s.KeysNS("", walkerID)
}

// --- Namespace-aware methods ---

func (s *InMemoryStore) GetNS(namespace, walkerID, key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns := s.data[namespace]
	if ns == nil {
		return nil, false
	}
	wk := ns[walkerID]
	if wk == nil {
		return nil, false
	}
	item, ok := wk[key]
	if !ok {
		return nil, false
	}
	return item.Value, true
}

func (s *InMemoryStore) SetNS(namespace, walkerID, key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[namespace] == nil {
		s.data[namespace] = make(map[string]map[string]MemoryItem)
	}
	if s.data[namespace][walkerID] == nil {
		s.data[namespace][walkerID] = make(map[string]MemoryItem)
	}
	s.data[namespace][walkerID][key] = MemoryItem{
		Namespace: namespace,
		WalkerID:  walkerID,
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
	}
}

func (s *InMemoryStore) KeysNS(namespace, walkerID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns := s.data[namespace]
	if ns == nil {
		return nil
	}
	wk := ns[walkerID]
	if wk == nil {
		return nil
	}
	keys := make([]string, 0, len(wk))
	for k := range wk {
		keys = append(keys, k)
	}
	return keys
}

// Search does substring matching on keys and string values across all walkers
// in the given namespace.
func (s *InMemoryStore) Search(namespace, query string) []MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns := s.data[namespace]
	if ns == nil {
		return nil
	}
	lower := strings.ToLower(query)
	var results []MemoryItem
	for _, wk := range ns {
		for _, item := range wk {
			if strings.Contains(strings.ToLower(item.Key), lower) {
				results = append(results, item)
				continue
			}
			if sv, ok := item.Value.(string); ok && strings.Contains(strings.ToLower(sv), lower) {
				results = append(results, item)
				continue
			}
			for _, tag := range item.Tags {
				if strings.Contains(strings.ToLower(tag), lower) {
					results = append(results, item)
					break
				}
			}
		}
	}
	return results
}

// SetNSTagged is like SetNS but also attaches tags to the memory item.
func (s *InMemoryStore) SetNSTagged(namespace, walkerID, key string, value any, tags []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[namespace] == nil {
		s.data[namespace] = make(map[string]map[string]MemoryItem)
	}
	if s.data[namespace][walkerID] == nil {
		s.data[namespace][walkerID] = make(map[string]MemoryItem)
	}
	s.data[namespace][walkerID][key] = MemoryItem{
		Namespace: namespace,
		WalkerID:  walkerID,
		Key:       key,
		Value:     value,
		Tags:      tags,
		CreatedAt: time.Now(),
	}
}

// --- Memory type helper functions ---

// SetFact stores a semantic fact about a walker.
func SetFact(store MemoryStore, walkerID, key string, value any) {
	store.SetNS(NamespaceSemantic, walkerID, key, value)
}

// RecordEpisode stores an episodic memory (a walk summary).
func RecordEpisode(store MemoryStore, walkerID, walkID string, summary string) {
	store.SetNS(NamespaceEpisodic, walkerID, walkID, summary)
}

// UpdateInstruction stores a procedural memory (a prompt refinement).
func UpdateInstruction(store MemoryStore, walkerID, key string, instruction string) {
	store.SetNS(NamespaceProcedural, walkerID, key, instruction)
}
