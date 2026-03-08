package framework

// Category: Processing & Support

import (
	"context"
	"sync"
	"time"
)

// FindingSeverity classifies the impact level of an enforcer finding.
type FindingSeverity string

const (
	FindingInfo    FindingSeverity = "info"
	FindingWarning FindingSeverity = "warning"
	FindingError   FindingSeverity = "error"
)

var severityOrder = map[FindingSeverity]int{
	FindingInfo:    0,
	FindingWarning: 1,
	FindingError:   2,
}

// SeverityAtOrAbove returns true when have is at or above the threshold severity.
func SeverityAtOrAbove(have, threshold FindingSeverity) bool {
	return severityOrder[have] >= severityOrder[threshold]
}

// Finding is a typed observation produced by an enforcer during circuit execution.
// All three enforcement patterns (Hook, Signal, Parallel Circuit) produce the same type.
type Finding struct {
	Severity  FindingSeverity `json:"severity"`
	Domain    string          `json:"domain"`
	Source    string          `json:"source"`
	NodeName  string          `json:"node_name"`
	Message   string          `json:"message"`
	Evidence  map[string]any  `json:"evidence,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// FindingCollector accumulates findings during a walk.
type FindingCollector interface {
	Report(ctx context.Context, f Finding) error
	Findings() []Finding
}

// FindingCollectorKey is the well-known key used to store a FindingCollector
// in WalkerState.Context, making it available to expression edges and hooks.
const FindingCollectorKey = "__finding_collector"

// InMemoryFindingCollector is a thread-safe, slice-backed FindingCollector.
type InMemoryFindingCollector struct {
	mu       sync.RWMutex
	findings []Finding
}

func (c *InMemoryFindingCollector) Report(_ context.Context, f Finding) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if f.Timestamp.IsZero() {
		f.Timestamp = time.Now().UTC()
	}
	c.findings = append(c.findings, f)
	return nil
}

func (c *InMemoryFindingCollector) Findings() []Finding {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Finding, len(c.findings))
	copy(out, c.findings)
	return out
}

// vetoArtifact wraps an artifact and overrides Confidence to 0.
// Used by the hookingWalker when a VetoHook returns ErrFindingVeto.
type vetoArtifact struct {
	inner Artifact
}

func (v *vetoArtifact) Type() string       { return v.inner.Type() }
func (v *vetoArtifact) Confidence() float64 { return 0 }
func (v *vetoArtifact) Raw() any            { return v.inner.Raw() }
