package framework

// Category: Processing & Support

import "context"

// VetoHook is an after-hook that checks the FindingCollector for
// FindingError findings targeting the current node. When found, it
// returns ErrFindingVeto, which the hookingWalker intercepts to wrap
// the artifact with Confidence() 0.
type VetoHook struct {
	collector FindingCollector
}

// NewVetoHook creates a VetoHook backed by the given collector.
func NewVetoHook(collector FindingCollector) *VetoHook {
	return &VetoHook{collector: collector}
}

func (h *VetoHook) Name() string { return "finding-veto" }

func (h *VetoHook) Run(_ context.Context, nodeName string, artifact Artifact) error {
	if artifact == nil {
		return nil
	}
	for _, f := range h.collector.Findings() {
		if f.Severity == FindingError && f.NodeName == nodeName {
			return ErrFindingVeto
		}
	}
	return nil
}
