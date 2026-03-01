package ouroboros

import (
	"strings"
	"testing"
)

func TestCompensate_OverconfidentModel(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimEvidenceDepth: 0.85,
		DimSpeed:         0.5,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "overstate confidence") {
		t.Errorf("expected overconfidence correction, got: %s", result)
	}
}

func TestCompensate_HastyModel(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimSpeed:         0.85,
		DimEvidenceDepth: 0.2,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "rush analysis") {
		t.Errorf("expected rush correction, got: %s", result)
	}
}

func TestCompensate_ShortcutProne(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimShortcutAffinity: 0.9,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "take shortcuts") {
		t.Errorf("expected shortcut correction, got: %s", result)
	}
}

func TestCompensate_LowPersistence(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimPersistence: 0.1,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "give up") {
		t.Errorf("expected persistence correction, got: %s", result)
	}
}

func TestCompensate_PrematureConvergence(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimConvergenceThreshold: 0.15,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "accept conclusions prematurely") {
		t.Errorf("expected convergence correction, got: %s", result)
	}
}

func TestCompensate_VerboseModel(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimFailureMode: 0.85,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "verbose") {
		t.Errorf("expected verbosity correction, got: %s", result)
	}
}

func TestCompensate_NoBias(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimSpeed:                0.5,
		DimEvidenceDepth:        0.5,
		DimPersistence:          0.5,
		DimShortcutAffinity:     0.5,
		DimConvergenceThreshold: 0.5,
		DimFailureMode:          0.5,
	}
	result := c.Compensate(dims)
	if result != "" {
		t.Errorf("expected empty correction for balanced model, got: %s", result)
	}
}

func TestCompensate_EmptyDimensions(t *testing.T) {
	c := DefaultOffsetCompensator()
	result := c.Compensate(nil)
	if result != "" {
		t.Errorf("expected empty for nil, got: %s", result)
	}
}

func TestCompensate_MultipleBiases(t *testing.T) {
	c := DefaultOffsetCompensator()
	dims := map[Dimension]float64{
		DimEvidenceDepth:    0.9,
		DimShortcutAffinity: 0.8,
		DimPersistence:      0.1,
	}
	result := c.Compensate(dims)

	if !strings.Contains(result, "overstate confidence") {
		t.Error("missing overconfidence correction")
	}
	if !strings.Contains(result, "take shortcuts") {
		t.Error("missing shortcut correction")
	}
	if !strings.Contains(result, "give up") {
		t.Error("missing persistence correction")
	}
	if !strings.Contains(result, "[Calibration offset") {
		t.Error("missing header")
	}
}

func TestCompensate_CustomThresholds(t *testing.T) {
	c := &OffsetCompensator{
		HighThreshold: 0.6,
		LowThreshold:  0.4,
	}
	dims := map[Dimension]float64{
		DimEvidenceDepth: 0.65,
	}
	result := c.Compensate(dims)
	if !strings.Contains(result, "overstate confidence") {
		t.Errorf("custom threshold should trigger at 0.65, got: %s", result)
	}
}
