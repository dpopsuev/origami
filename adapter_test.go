package framework

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAdapterManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "adapter.yaml")

	manifest := `
adapter: test-adapter
namespace: test
version: "1.0.0"
description: A test adapter
provides:
  transformers: [my-transform]
  extractors: [my-extract]
  hooks: [my-hook]
requires:
  origami: ">=0.1.0"
`
	if err := os.WriteFile(path, []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadAdapterManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if m.Namespace != "test" {
		t.Errorf("namespace = %q, want %q", m.Namespace, "test")
	}
	if m.Adapter != "test-adapter" {
		t.Errorf("adapter = %q, want %q", m.Adapter, "test-adapter")
	}
	if m.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", m.Version, "1.0.0")
	}
	if len(m.Provides.Transformers) != 1 || m.Provides.Transformers[0] != "my-transform" {
		t.Errorf("provides.transformers = %v, want [my-transform]", m.Provides.Transformers)
	}
	if len(m.Provides.Extractors) != 1 || m.Provides.Extractors[0] != "my-extract" {
		t.Errorf("provides.extractors = %v, want [my-extract]", m.Provides.Extractors)
	}
	if len(m.Provides.Hooks) != 1 || m.Provides.Hooks[0] != "my-hook" {
		t.Errorf("provides.hooks = %v, want [my-hook]", m.Provides.Hooks)
	}
}

func TestLoadAdapterManifest_MissingNamespace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "adapter.yaml")
	if err := os.WriteFile(path, []byte("adapter: bad\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAdapterManifest(path)
	if err == nil {
		t.Fatal("expected error for missing namespace")
	}
}

func TestMergeAdapters_NoCollision(t *testing.T) {
	stubT := TransformerFunc("base-t", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "base", nil
	})
	base := GraphRegistries{
		Transformers: TransformerRegistry{"base-t": stubT},
		Extractors:   ExtractorRegistry{},
		Hooks:        HookRegistry{},
	}

	adapterT := TransformerFunc("ext-t", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "adapter", nil
	})
	adapter := &Adapter{
		Namespace:    "vendor",
		Transformers: TransformerRegistry{"ext-t": adapterT},
	}

	merged, err := MergeAdapters(base, adapter)
	if err != nil {
		t.Fatal(err)
	}

	// FQCN key present
	if _, err := merged.Transformers.Get("vendor.ext-t"); err != nil {
		t.Errorf("FQCN lookup failed: %v", err)
	}
	// Short name present (no collision with base)
	if _, err := merged.Transformers.Get("ext-t"); err != nil {
		t.Errorf("short name lookup failed: %v", err)
	}
	// Base still present
	if _, err := merged.Transformers.Get("base-t"); err != nil {
		t.Errorf("base lookup failed: %v", err)
	}
}

func TestMergeAdapters_ShortNameCollision(t *testing.T) {
	stubT := TransformerFunc("llm", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "base-llm", nil
	})
	base := GraphRegistries{
		Transformers: TransformerRegistry{"llm": stubT},
		Extractors:   ExtractorRegistry{},
		Hooks:        HookRegistry{},
	}

	adapterT := TransformerFunc("llm", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "adapter-llm", nil
	})
	adapter := &Adapter{
		Namespace:    "custom",
		Transformers: TransformerRegistry{"llm": adapterT},
	}

	merged, err := MergeAdapters(base, adapter)
	if err != nil {
		t.Fatal("MergeAdapters should succeed; short name collision is not fatal")
	}

	// FQCN resolves to adapter's version
	if _, err := merged.Transformers.Get("custom.llm"); err != nil {
		t.Errorf("FQCN lookup failed: %v", err)
	}
	// Short name still resolves to the base (first-registered wins)
	result, err := merged.Transformers.Get("llm")
	if err != nil {
		t.Fatal(err)
	}
	out, _ := result.Transform(context.Background(), &TransformerContext{})
	if out != "base-llm" {
		t.Errorf("short name should resolve to base; got %v", out)
	}
}

func TestMergeAdapters_FQCNCollision(t *testing.T) {
	base := GraphRegistries{
		Transformers: TransformerRegistry{},
		Extractors:   ExtractorRegistry{},
		Hooks:        HookRegistry{},
	}

	t1 := TransformerFunc("llm", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "a1", nil
	})
	t2 := TransformerFunc("llm", func(_ context.Context, _ *TransformerContext) (any, error) {
		return "a2", nil
	})
	a1 := &Adapter{Namespace: "vendor", Transformers: TransformerRegistry{"llm": t1}}
	a2 := &Adapter{Namespace: "vendor", Transformers: TransformerRegistry{"llm": t2}}

	_, err := MergeAdapters(base, a1, a2)
	if err == nil {
		t.Fatal("expected FQCN collision error")
	}
	if got := err.Error(); got != `transformer "vendor.llm" collision (adapter vendor)` {
		t.Errorf("unexpected error: %v", got)
	}
}

func TestMergeAdapters_DoesNotMutateBase(t *testing.T) {
	base := GraphRegistries{
		Transformers: TransformerRegistry{
			"base-t": TransformerFunc("base-t", func(_ context.Context, _ *TransformerContext) (any, error) { return nil, nil }),
		},
		Extractors: ExtractorRegistry{},
		Hooks:      HookRegistry{},
	}

	adapter := &Adapter{
		Namespace: "x",
		Transformers: TransformerRegistry{
			"new-t": TransformerFunc("new-t", func(_ context.Context, _ *TransformerContext) (any, error) { return nil, nil }),
		},
	}

	_, err := MergeAdapters(base, adapter)
	if err != nil {
		t.Fatal(err)
	}

	// Base must be unchanged
	if len(base.Transformers) != 1 {
		t.Errorf("base.Transformers was mutated: %d entries, want 1", len(base.Transformers))
	}
}

func TestResolveFQCN(t *testing.T) {
	tests := []struct {
		input     string
		wantNS    string
		wantName  string
	}{
		{"vendor.llm", "vendor", "llm"},
		{"llm", "", "llm"},
		{"a.b.c", "a", "b.c"},
		{".leading", "", ".leading"},
	}
	for _, tt := range tests {
		ns, name := ResolveFQCN(tt.input)
		if ns != tt.wantNS || name != tt.wantName {
			t.Errorf("ResolveFQCN(%q) = (%q, %q), want (%q, %q)",
				tt.input, ns, name, tt.wantNS, tt.wantName)
		}
	}
}

func TestMergeAdapters_Hooks(t *testing.T) {
	base := GraphRegistries{
		Transformers: TransformerRegistry{},
		Extractors:   ExtractorRegistry{},
		Hooks:        HookRegistry{},
	}

	hook := NewHookFunc("store", func(_ context.Context, _ string, _ Artifact) error { return nil })
	adapter := &Adapter{
		Namespace: "rca",
		Hooks:     HookRegistry{"store": hook},
	}

	merged, err := MergeAdapters(base, adapter)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := merged.Hooks.Get("rca.store"); err != nil {
		t.Errorf("FQCN hook lookup failed: %v", err)
	}
	if _, err := merged.Hooks.Get("store"); err != nil {
		t.Errorf("short name hook lookup failed: %v", err)
	}
}

func TestMergeAdapters_Extractors(t *testing.T) {
	base := GraphRegistries{
		Transformers: TransformerRegistry{},
		Extractors:   ExtractorRegistry{},
		Hooks:        HookRegistry{},
	}

	ext := &adapterStubExtractor{name: "govulncheck"}
	adapter := &Adapter{
		Namespace:  "achilles",
		Extractors: ExtractorRegistry{"govulncheck": ext},
	}

	merged, err := MergeAdapters(base, adapter)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := merged.Extractors.Get("achilles.govulncheck"); err != nil {
		t.Errorf("FQCN extractor lookup failed: %v", err)
	}
	if _, err := merged.Extractors.Get("govulncheck"); err != nil {
		t.Errorf("short name extractor lookup failed: %v", err)
	}
}

type adapterStubExtractor struct {
	name string
}

func (e *adapterStubExtractor) Name() string                                   { return e.name }
func (e *adapterStubExtractor) Extract(_ context.Context, _ any) (any, error) { return "extracted", nil }

func TestPipelineDef_ImportsField(t *testing.T) {
	yaml := `
pipeline: test
imports:
  - vendor.rca-tools
  - vendor.vuln-tools
nodes:
  - name: start
edges:
  - id: e1
    from: start
    to: done
start: start
done: done
`
	def, err := LoadPipeline([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Imports) != 2 {
		t.Fatalf("imports = %v, want 2 entries", def.Imports)
	}
	if def.Imports[0] != "vendor.rca-tools" {
		t.Errorf("imports[0] = %q, want %q", def.Imports[0], "vendor.rca-tools")
	}
	if def.Imports[1] != "vendor.vuln-tools" {
		t.Errorf("imports[1] = %q, want %q", def.Imports[1], "vendor.vuln-tools")
	}
}
