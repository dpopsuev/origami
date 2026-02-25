package framework

import (
	"sort"
	"sync"
	"testing"
)

func TestInMemoryStoreSetAndGet(t *testing.T) {
	store := NewInMemoryStore()
	store.Set("walker-1", "key-a", "value-a")

	got, ok := store.Get("walker-1", "key-a")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if got != "value-a" {
		t.Errorf("Get = %v, want %q", got, "value-a")
	}
}

func TestInMemoryStoreGetMissing(t *testing.T) {
	store := NewInMemoryStore()

	_, ok := store.Get("nonexistent", "key")
	if ok {
		t.Error("expected missing walker to return false")
	}

	store.Set("walker-1", "key-a", "value")
	_, ok = store.Get("walker-1", "key-b")
	if ok {
		t.Error("expected missing key to return false")
	}
}

func TestInMemoryStoreIsolation(t *testing.T) {
	store := NewInMemoryStore()
	store.Set("walker-1", "shared-key", "value-1")
	store.Set("walker-2", "shared-key", "value-2")

	v1, _ := store.Get("walker-1", "shared-key")
	v2, _ := store.Get("walker-2", "shared-key")

	if v1 != "value-1" {
		t.Errorf("walker-1 value = %v, want %q", v1, "value-1")
	}
	if v2 != "value-2" {
		t.Errorf("walker-2 value = %v, want %q", v2, "value-2")
	}
}

func TestInMemoryStoreKeys(t *testing.T) {
	store := NewInMemoryStore()
	store.Set("w1", "b", 2)
	store.Set("w1", "a", 1)
	store.Set("w1", "c", 3)

	keys := store.Keys("w1")
	sort.Strings(keys)

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("keys = %v, want [a b c]", keys)
	}
}

func TestInMemoryStoreKeysEmpty(t *testing.T) {
	store := NewInMemoryStore()
	keys := store.Keys("nonexistent")
	if keys != nil {
		t.Errorf("expected nil for nonexistent walker, got %v", keys)
	}
}

func TestInMemoryStoreConcurrentSafety(t *testing.T) {
	store := NewInMemoryStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			walkerID := "walker"
			key := "counter"
			store.Set(walkerID, key, n)
			store.Get(walkerID, key)
			store.Keys(walkerID)
		}(i)
	}

	wg.Wait()

	_, ok := store.Get("walker", "counter")
	if !ok {
		t.Error("expected key to exist after concurrent writes")
	}
}

func TestInMemoryStorePersistsAcrossReads(t *testing.T) {
	store := NewInMemoryStore()

	store.Set("agent-a", "step_count", 5)

	got, ok := store.Get("agent-a", "step_count")
	if !ok || got != 5 {
		t.Errorf("first read: got=%v ok=%v, want 5/true", got, ok)
	}

	store.Set("agent-a", "step_count", 10)

	got, ok = store.Get("agent-a", "step_count")
	if !ok || got != 10 {
		t.Errorf("second read after update: got=%v ok=%v, want 10/true", got, ok)
	}
}

func TestWithMemoryRunOption(t *testing.T) {
	store := NewInMemoryStore()
	opt := WithMemory(store)

	cfg := &runConfig{}
	opt(cfg)

	if cfg.memory != store {
		t.Error("WithMemory did not set memory on runConfig")
	}
}
