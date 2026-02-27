package knowledge_test

import (
	"strings"
	"testing"

	"github.com/dpopsuev/origami/knowledge"
)

func TestTruncateSummarizer_Full_FitsInBudget(t *testing.T) {
	s := knowledge.TruncateSummarizer{}
	content := "short content"
	got := s.Summarize(content, 100, knowledge.StrategyFull)
	if got != content {
		t.Errorf("expected unmodified content, got %q", got)
	}
}

func TestTruncateSummarizer_Full_Truncated(t *testing.T) {
	s := knowledge.TruncateSummarizer{}
	content := strings.Repeat("x", 1000)
	got := s.Summarize(content, 50, knowledge.StrategyFull)
	if !strings.Contains(got, "[truncated to fit token budget]") {
		t.Error("expected truncation marker")
	}
	if len(got) > 250 {
		t.Errorf("got %d chars, expected ~200 + marker", len(got))
	}
}

func TestTruncateSummarizer_Summary(t *testing.T) {
	s := knowledge.TruncateSummarizer{}
	content := strings.Repeat("a", 500) + strings.Repeat("b", 500)
	got := s.Summarize(content, 50, knowledge.StrategySummary)
	if !strings.Contains(got, "[middle omitted]") {
		t.Error("expected middle omission marker")
	}
}

func TestTruncateSummarizer_OnDemand(t *testing.T) {
	s := knowledge.TruncateSummarizer{}
	got := s.Summarize("hello world", 10, knowledge.StrategyOnDemand)
	if !strings.Contains(got, "available on demand") {
		t.Errorf("expected on-demand message, got %q", got)
	}
}

func TestTruncateSummarizer_IndexOnly(t *testing.T) {
	s := knowledge.TruncateSummarizer{}
	content := "line1\nline2\nline3"
	got := s.Summarize(content, 10, knowledge.StrategyIndexOnly)
	if !strings.Contains(got, "3 lines") {
		t.Errorf("expected line count, got %q", got)
	}
}

func TestBudgetAllocator_Allocate(t *testing.T) {
	ba := knowledge.BudgetAllocator{TotalBudget: 1000}
	sources := []knowledge.Source{
		{Name: "always", ReadPolicy: knowledge.ReadAlways},
		{Name: "conditional", ReadPolicy: knowledge.ReadConditional},
	}
	entries := ba.Allocate(sources)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Strategy != knowledge.StrategyFull {
		t.Errorf("always-read should get StrategyFull, got %s", entries[0].Strategy)
	}
	if entries[1].Strategy != knowledge.StrategySummary {
		t.Errorf("conditional should get StrategySummary, got %s", entries[1].Strategy)
	}
	if entries[0].Budget != 500 {
		t.Errorf("expected 500 per source, got %d", entries[0].Budget)
	}
}

func TestBudgetAllocator_Empty(t *testing.T) {
	ba := knowledge.BudgetAllocator{TotalBudget: 1000}
	entries := ba.Allocate(nil)
	if entries != nil {
		t.Error("expected nil for empty sources")
	}
}
