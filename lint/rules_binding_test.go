package lint

import (
	"testing"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/fold"
)

func makeManifestCtx(bindings map[string]string, comps map[string]*framework.ComponentManifest) *ManifestLintContext {
	m := &fold.Manifest{
		Name:     "test",
		Imports:  []string{"origami.schematics.rca"},
		Bindings: bindings,
	}
	return &ManifestLintContext{
		Manifest:   m,
		Registry:   fold.DefaultRegistry(),
		File:       "origami.yaml",
		Components: comps,
	}
}

func compWithSockets(sockets ...framework.SocketDef) *framework.ComponentManifest {
	c := &framework.ComponentManifest{Component: "test"}
	c.Requires.Sockets = sockets
	return c
}

func compWithSatisfies(satisfies ...framework.SatisfiesDef) *framework.ComponentManifest {
	c := &framework.ComponentManifest{Component: "test"}
	c.Satisfies = satisfies
	return c
}

func TestUnboundSocket_Finding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.schematics.rca": compWithSockets(
			framework.SocketDef{Name: "store", Type: "store.Store"},
			framework.SocketDef{Name: "source", Type: "SourceReader"},
		),
	}
	ctx := makeManifestCtx(map[string]string{"source": "origami.connectors.rp"}, comps)

	findings := (&UnboundSocket{}).CheckManifest(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].RuleID != "M1/unbound-socket" {
		t.Errorf("rule ID = %q", findings[0].RuleID)
	}
}

func TestUnboundSocket_NoFinding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.schematics.rca": compWithSockets(
			framework.SocketDef{Name: "source", Type: "SourceReader"},
		),
	}
	ctx := makeManifestCtx(map[string]string{"source": "origami.connectors.rp"}, comps)

	findings := (&UnboundSocket{}).CheckManifest(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %v", len(findings), findings)
	}
}

func TestUnknownBinding_Finding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.schematics.rca": compWithSockets(
			framework.SocketDef{Name: "source", Type: "SourceReader"},
		),
	}
	ctx := makeManifestCtx(map[string]string{
		"source":   "origami.connectors.rp",
		"nonexist": "origami.connectors.foo",
	}, comps)

	findings := (&UnknownBinding{}).CheckManifest(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %v", len(findings), findings)
	}
	if findings[0].RuleID != "M2/unknown-binding" {
		t.Errorf("rule ID = %q", findings[0].RuleID)
	}
}

func TestUnknownBinding_NoFinding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.schematics.rca": compWithSockets(
			framework.SocketDef{Name: "source", Type: "SourceReader"},
		),
	}
	ctx := makeManifestCtx(map[string]string{"source": "origami.connectors.rp"}, comps)

	findings := (&UnknownBinding{}).CheckManifest(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %v", len(findings), findings)
	}
}

func TestUnsatisfiedConnector_Finding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.connectors.rp": compWithSatisfies(
			framework.SatisfiesDef{Socket: "other", Factory: "NewOther"},
		),
	}
	ctx := makeManifestCtx(map[string]string{"source": "origami.connectors.rp"}, comps)

	findings := (&UnsatisfiedConnector{}).CheckManifest(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %v", len(findings), findings)
	}
	if findings[0].RuleID != "M3/unsatisfied-connector" {
		t.Errorf("rule ID = %q", findings[0].RuleID)
	}
}

func TestUnsatisfiedConnector_NoFinding(t *testing.T) {
	comps := map[string]*framework.ComponentManifest{
		"origami.connectors.rp": compWithSatisfies(
			framework.SatisfiesDef{Socket: "source", Factory: "NewSourceReader"},
		),
	}
	ctx := makeManifestCtx(map[string]string{"source": "origami.connectors.rp"}, comps)

	findings := (&UnsatisfiedConnector{}).CheckManifest(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %v", len(findings), findings)
	}
}

func TestMissingConnector_Finding(t *testing.T) {
	ctx := makeManifestCtx(map[string]string{
		"source": "unknown.connectors.rp",
	}, map[string]*framework.ComponentManifest{})

	findings := (&MissingConnector{}).CheckManifest(ctx)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %v", len(findings), findings)
	}
	if findings[0].RuleID != "M4/missing-connector" {
		t.Errorf("rule ID = %q", findings[0].RuleID)
	}
}

func TestMissingConnector_NoFinding(t *testing.T) {
	ctx := makeManifestCtx(map[string]string{
		"source": "origami.connectors.rp",
	}, map[string]*framework.ComponentManifest{})

	findings := (&MissingConnector{}).CheckManifest(ctx)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %v", len(findings), findings)
	}
}
