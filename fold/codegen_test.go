package fold

import (
	"strings"
	"testing"
)

// --- Multi-Schematic Composition Tests ---

func TestGenerateMain_WithSecondarySchematic_InProcess(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		Imports: []string{
			"origami.schematics.rca",
			"origami.schematics.knowledge",
		},
		Bindings: map[string]string{
			"rca.source":    "origami.connectors.rp",
			"knowledge.git":  "origami.connectors.github",
			"knowledge.docs": "origami.connectors.docs",
		},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)

	// Should have the knowledge package import
	if !strings.Contains(code, `"github.com/dpopsuev/origami/schematics/knowledge"`) {
		t.Errorf("missing knowledge import in:\n%s", code)
	}

	// Should construct the secondary schematic
	if !strings.Contains(code, "knowledge.NewRouter(") {
		t.Errorf("missing secondary construction in:\n%s", code)
	}

	// Should wire git and docs drivers to the secondary
	if !strings.Contains(code, "knowledge.WithGitDriver(") {
		t.Errorf("missing WithGitDriver option in:\n%s", code)
	}
	if !strings.Contains(code, "knowledge.WithDocsDriver(") {
		t.Errorf("missing WithDocsDriver option in:\n%s", code)
	}

	// Should pass the secondary to the primary's Apply
	if !strings.Contains(code, "WithKnowledgeReader(knowledgeSchematic)") {
		t.Errorf("missing WithKnowledgeReader injection in:\n%s", code)
	}

	// Should NOT import subprocess for in-process mode
	if strings.Contains(code, "subprocess") {
		t.Errorf("unexpected subprocess import for in-process mode in:\n%s", code)
	}
}

func TestGenerateMain_WithSecondarySchematic_Subprocess(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		Imports: []string{
			"origami.schematics.rca",
			"origami.schematics.knowledge",
		},
		Bindings: map[string]string{
			"rca.source":    "origami.connectors.rp",
			"knowledge.git":  "origami.connectors.github",
			"knowledge.docs": "origami.connectors.docs",
		},
		Deploy: map[string]*DeployConfig{
			"knowledge": {Mode: "subprocess"},
		},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)

	// Should import subprocess package
	if !strings.Contains(code, `"github.com/dpopsuev/origami/subprocess"`) {
		t.Errorf("missing subprocess import in:\n%s", code)
	}

	// Should import context and log for subprocess startup
	if !strings.Contains(code, `"context"`) {
		t.Errorf("missing context import in:\n%s", code)
	}

	// Should create subprocess.Server
	if !strings.Contains(code, "subprocess.Server{") {
		t.Errorf("missing subprocess.Server construction in:\n%s", code)
	}

	// Should start and defer stop
	if !strings.Contains(code, ".Start(context.Background())") {
		t.Errorf("missing subprocess Start in:\n%s", code)
	}
	if !strings.Contains(code, ".Stop(context.Background())") {
		t.Errorf("missing subprocess Stop in:\n%s", code)
	}

	// Should pass the subprocess server to the primary
	if !strings.Contains(code, "WithKnowledgeReader(knowledgeSchematicSrv)") {
		t.Errorf("missing WithKnowledgeReader subprocess injection in:\n%s", code)
	}
}

func TestGenerateMain_SecondarySchematicNotInImports(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		Imports: []string{"origami.schematics.rca"},
		// knowledge bindings present but knowledge schematic not imported —
		// should be treated as regular bindings (will fail socket lookup)
		Bindings: map[string]string{
			"rca.source": "origami.connectors.rp",
		},
	}

	// Should succeed — knowledge schematic socket is optional when not imported
	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)

	// Should NOT have knowledge schematic construction
	if strings.Contains(code, "knowledge.NewRouter") {
		t.Errorf("unexpected knowledge construction when not imported:\n%s", code)
	}
}

func TestGenerateMain(t *testing.T) {
	m := &Manifest{
		Name:        "asterisk",
		Description: "Evidence-based RCA",
		Version:     "1.0",
		Imports:     []string{"origami.schematics.rca"},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}

	code := string(src)

	if !strings.Contains(code, "DO NOT EDIT") {
		t.Error("missing DO NOT EDIT comment")
	}
	if !strings.Contains(code, `"github.com/dpopsuev/origami/schematics/rca/cmd"`) {
		t.Errorf("missing cmd import in:\n%s", code)
	}
	if !strings.Contains(code, ".Execute()") {
		t.Errorf("missing Execute() call in:\n%s", code)
	}
	if !strings.Contains(code, "package main") {
		t.Errorf("missing package main in:\n%s", code)
	}
}

func TestGenerateMain_NoImports(t *testing.T) {
	m := &Manifest{
		Name: "empty",
	}
	_, err := GenerateMain(m, DefaultRegistry())
	if err == nil {
		t.Fatal("expected error for no imports")
	}
}

func TestGenerateMain_WithBindings(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		Imports: []string{"origami.schematics.rca"},
		Bindings: map[string]string{
			"source": "origami.connectors.rp",
		},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}

	code := string(src)

	if !strings.Contains(code, `rp "github.com/dpopsuev/origami/connectors/rp"`) {
		t.Errorf("missing rp connector import in:\n%s", code)
	}
	if !strings.Contains(code, ".Apply(") {
		t.Errorf("missing Apply() call in:\n%s", code)
	}
	if !strings.Contains(code, "WithSourceReader") {
		t.Errorf("missing WithSourceReader option in:\n%s", code)
	}
	if !strings.Contains(code, "rp.NewSourceReader") {
		t.Errorf("missing rp.NewSourceReader factory in:\n%s", code)
	}
}

func TestGenerateMain_WithStoreSchema(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		Imports: []string{"origami.schematics.rca"},
		Bindings: map[string]string{
			"source":       "origami.connectors.rp",
			"store.schema": "schema.yaml",
		},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}

	code := string(src)

	if !strings.Contains(code, `_ "embed"`) {
		t.Errorf("missing embed import in:\n%s", code)
	}
	if !strings.Contains(code, "//go:embed schema.yaml") {
		t.Errorf("missing go:embed directive in:\n%s", code)
	}
	if !strings.Contains(code, "var storeSchema []byte") {
		t.Errorf("missing storeSchema var in:\n%s", code)
	}
	if !strings.Contains(code, "WithStoreSchema(storeSchema)") {
		t.Errorf("missing WithStoreSchema call in:\n%s", code)
	}
	if !strings.Contains(code, "WithSourceReader") {
		t.Errorf("store.schema should not suppress socket bindings in:\n%s", code)
	}
}

func TestGenerateMain_StoreSchemaOnly(t *testing.T) {
	m := &Manifest{
		Name:    "myapp",
		Version: "1.0",
		Imports: []string{"origami.schematics.rca"},
		Bindings: map[string]string{
			"store.schema": "db/schema.yaml",
		},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}

	code := string(src)

	if !strings.Contains(code, "//go:embed schema.yaml") {
		t.Errorf("expected base filename in embed directive, got:\n%s", code)
	}
	if !strings.Contains(code, ".Apply(") {
		t.Errorf("WithStoreSchema should trigger Apply in:\n%s", code)
	}
}
