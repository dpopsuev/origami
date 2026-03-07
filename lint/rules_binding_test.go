package lint

import (
	"testing"

	"github.com/dpopsuev/origami/fold"
)

func TestRunManifestRules_Empty(t *testing.T) {
	m := &fold.Manifest{
		Name:    "test",
		Version: "1.0",
	}
	ctx := NewManifestLintContext(m, "origami.yaml")
	findings := RunManifestRules(ctx)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty rules, got %d", len(findings))
	}
}
