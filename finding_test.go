package framework

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestFindingSeverity_Constants(t *testing.T) {
	if FindingInfo != "info" {
		t.Errorf("FindingInfo = %q, want %q", FindingInfo, "info")
	}
	if FindingWarning != "warning" {
		t.Errorf("FindingWarning = %q, want %q", FindingWarning, "warning")
	}
	if FindingError != "error" {
		t.Errorf("FindingError = %q, want %q", FindingError, "error")
	}
}

func TestSeverityAtOrAbove(t *testing.T) {
	tests := []struct {
		have      FindingSeverity
		threshold FindingSeverity
		want      bool
	}{
		{FindingInfo, FindingInfo, true},
		{FindingWarning, FindingInfo, true},
		{FindingError, FindingInfo, true},
		{FindingInfo, FindingWarning, false},
		{FindingWarning, FindingWarning, true},
		{FindingError, FindingWarning, true},
		{FindingInfo, FindingError, false},
		{FindingWarning, FindingError, false},
		{FindingError, FindingError, true},
	}
	for _, tt := range tests {
		got := SeverityAtOrAbove(tt.have, tt.threshold)
		if got != tt.want {
			t.Errorf("SeverityAtOrAbove(%q, %q) = %v, want %v", tt.have, tt.threshold, got, tt.want)
		}
	}
}

func TestFinding_Construction(t *testing.T) {
	ts := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	f := Finding{
		Severity:  FindingError,
		Domain:    "security.auth",
		Source:    "auth-enforcer",
		NodeName:  "login",
		Message:   "credentials exposed in artifact",
		Evidence:  map[string]any{"field": "password"},
		Timestamp: ts,
	}

	if f.Severity != FindingError {
		t.Errorf("Severity = %q, want %q", f.Severity, FindingError)
	}
	if f.Domain != "security.auth" {
		t.Errorf("Domain = %q, want %q", f.Domain, "security.auth")
	}
	if f.Evidence["field"] != "password" {
		t.Errorf("Evidence[field] = %v, want %q", f.Evidence["field"], "password")
	}
}

func TestInMemoryFindingCollector_Report(t *testing.T) {
	c := &InMemoryFindingCollector{}
	ctx := context.Background()

	f1 := Finding{Severity: FindingInfo, Domain: "lint", Source: "linter", Message: "style issue"}
	f2 := Finding{Severity: FindingWarning, Domain: "test", Source: "tester", Message: "flaky test"}

	if err := c.Report(ctx, f1); err != nil {
		t.Fatalf("Report f1: %v", err)
	}
	if err := c.Report(ctx, f2); err != nil {
		t.Fatalf("Report f2: %v", err)
	}

	findings := c.Findings()
	if len(findings) != 2 {
		t.Fatalf("len(Findings) = %d, want 2", len(findings))
	}
	if findings[0].Severity != FindingInfo {
		t.Errorf("findings[0].Severity = %q, want %q", findings[0].Severity, FindingInfo)
	}
	if findings[1].Severity != FindingWarning {
		t.Errorf("findings[1].Severity = %q, want %q", findings[1].Severity, FindingWarning)
	}
}

func TestInMemoryFindingCollector_TimestampDefault(t *testing.T) {
	c := &InMemoryFindingCollector{}
	before := time.Now().UTC()
	if err := c.Report(context.Background(), Finding{Severity: FindingInfo}); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()

	f := c.Findings()[0]
	if f.Timestamp.Before(before) || f.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in [%v, %v]", f.Timestamp, before, after)
	}
}

func TestInMemoryFindingCollector_FindingsReturnsCopy(t *testing.T) {
	c := &InMemoryFindingCollector{}
	_ = c.Report(context.Background(), Finding{Severity: FindingInfo, Message: "original"})

	findings := c.Findings()
	findings[0].Message = "mutated"

	if c.Findings()[0].Message != "original" {
		t.Error("Findings() did not return a copy; mutation leaked")
	}
}

func TestInMemoryFindingCollector_ConcurrentWrites(t *testing.T) {
	c := &InMemoryFindingCollector{}
	ctx := context.Background()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = c.Report(ctx, Finding{Severity: FindingInfo, Message: "concurrent"})
		}(i)
	}
	wg.Wait()

	if got := len(c.Findings()); got != n {
		t.Errorf("len(Findings) = %d, want %d", got, n)
	}
}

func TestVetoArtifact(t *testing.T) {
	inner := &findingStubArtifact{typ: "test", confidence: 0.95, raw: "data"}
	v := &vetoArtifact{inner: inner}

	if v.Type() != "test" {
		t.Errorf("Type() = %q, want %q", v.Type(), "test")
	}
	if v.Confidence() != 0 {
		t.Errorf("Confidence() = %f, want 0", v.Confidence())
	}
	if v.Raw() != "data" {
		t.Errorf("Raw() = %v, want %q", v.Raw(), "data")
	}
}

type findingStubArtifact struct {
	typ        string
	confidence float64
	raw        any
}

func (s *findingStubArtifact) Type() string       { return s.typ }
func (s *findingStubArtifact) Confidence() float64 { return s.confidence }
func (s *findingStubArtifact) Raw() any            { return s.raw }
