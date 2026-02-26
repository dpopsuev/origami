package framework

import (
	"sync"
	"testing"
	"time"
)

type stubCacheArtifact struct{ val string }

func (a *stubCacheArtifact) Type() string       { return "cache-test" }
func (a *stubCacheArtifact) Confidence() float64 { return 1.0 }
func (a *stubCacheArtifact) Raw() any            { return a.val }

func TestInMemoryCache_MissExecuteSet(t *testing.T) {
	c := NewInMemoryCache()

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	art := &stubCacheArtifact{val: "hello"}
	c.Set("key1", art, time.Minute)

	got, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected hit after set")
	}
	if got.(*stubCacheArtifact).val != "hello" {
		t.Errorf("got %v, want hello", got.Raw())
	}
}

func TestInMemoryCache_TTLExpiry(t *testing.T) {
	c := NewInMemoryCache()
	art := &stubCacheArtifact{val: "expire-me"}
	c.Set("key", art, time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestInMemoryCache_ZeroTTL_NeverExpires(t *testing.T) {
	c := NewInMemoryCache()
	art := &stubCacheArtifact{val: "forever"}
	c.Set("key", art, 0)

	got, ok := c.Get("key")
	if !ok || got.(*stubCacheArtifact).val != "forever" {
		t.Fatal("zero TTL should never expire")
	}
}

func TestInMemoryCache_Concurrency(t *testing.T) {
	c := NewInMemoryCache()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Set("k", &stubCacheArtifact{val: "v"}, time.Minute)
			c.Get("k")
		}()
	}

	wg.Wait()

	_, ok := c.Get("k")
	if !ok {
		t.Fatal("expected hit after concurrent writes")
	}
}

func TestInMemoryCache_Len(t *testing.T) {
	c := NewInMemoryCache()
	if c.Len() != 0 {
		t.Fatalf("want 0, got %d", c.Len())
	}
	c.Set("a", &stubCacheArtifact{val: "a"}, time.Minute)
	c.Set("b", &stubCacheArtifact{val: "b"}, time.Minute)
	if c.Len() != 2 {
		t.Fatalf("want 2, got %d", c.Len())
	}
}

func TestListArtifact(t *testing.T) {
	items := []Artifact{
		&stubCacheArtifact{val: "a"},
		&stubCacheArtifact{val: "b"},
	}
	la := &ListArtifact{Items: items}
	if la.Type() != "list" {
		t.Errorf("Type() = %q, want list", la.Type())
	}
	if la.Confidence() != 0 {
		t.Errorf("Confidence() = %f, want 0", la.Confidence())
	}
	raw := la.Raw().([]Artifact)
	if len(raw) != 2 {
		t.Errorf("len(Raw()) = %d, want 2", len(raw))
	}
}

func TestApplyMergeStrategy_Append(t *testing.T) {
	results := []branchResult{
		{nodeName: "a", artifact: &stubCacheArtifact{val: "a"}},
		{nodeName: "b", artifact: &stubCacheArtifact{val: "b"}},
	}
	merged := applyMergeStrategy(MergeAppend, results)
	la, ok := merged.(*ListArtifact)
	if !ok {
		t.Fatalf("expected *ListArtifact, got %T", merged)
	}
	if len(la.Items) != 2 {
		t.Errorf("want 2 items, got %d", len(la.Items))
	}
}

func TestApplyMergeStrategy_Latest(t *testing.T) {
	results := []branchResult{
		{nodeName: "a", artifact: &stubCacheArtifact{val: "first"}},
		{nodeName: "b", artifact: &stubCacheArtifact{val: "second"}},
	}
	merged := applyMergeStrategy(MergeLatest, results)
	if merged.(*stubCacheArtifact).val != "second" {
		t.Errorf("want second, got %v", merged.Raw())
	}
}

func TestApplyMergeStrategy_Custom(t *testing.T) {
	results := []branchResult{
		{nodeName: "a", artifact: &stubCacheArtifact{val: "first"}},
		{nodeName: "b", artifact: &stubCacheArtifact{val: "second"}},
	}
	merged := applyMergeStrategy(MergeCustom, results)
	if merged.(*stubCacheArtifact).val != "first" {
		t.Errorf("custom should return first, got %v", merged.Raw())
	}
}

func TestApplyMergeStrategy_Default(t *testing.T) {
	results := []branchResult{
		{nodeName: "a", artifact: &stubCacheArtifact{val: "first"}},
	}
	merged := applyMergeStrategy("", results)
	if merged.(*stubCacheArtifact).val != "first" {
		t.Errorf("default should return first, got %v", merged.Raw())
	}
}

func TestApplyMergeStrategy_Empty(t *testing.T) {
	merged := applyMergeStrategy(MergeAppend, nil)
	if merged != nil {
		t.Errorf("empty results should return nil, got %v", merged)
	}
}
