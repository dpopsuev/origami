package fold

import (
	"testing"
)

func TestParseManifest_Minimal(t *testing.T) {
	data := []byte(`
name: test-tool
description: A test tool
version: "1.0"
imports:
  - origami.modules.rca
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "test-tool" {
		t.Errorf("name = %q, want test-tool", m.Name)
	}
	if m.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", m.Version)
	}
	if len(m.Imports) != 1 {
		t.Fatalf("imports = %d, want 1", len(m.Imports))
	}
	if m.Imports[0] != "origami.modules.rca" {
		t.Errorf("imports[0] = %q", m.Imports[0])
	}
}

func TestParseManifest_Full(t *testing.T) {
	data := []byte(`
name: asterisk
description: Evidence-based RCA for ReportPortal test failures
version: "1.0"

imports:
  - origami.modules.rca
  - origami.adapters.rp
  - origami.adapters.sqlite

embed:
  - circuits/
  - scenarios/
  - prompts/

cli:
  global_flags:
    - {name: log-level, type: string, default: info}
    - {name: log-format, type: string, default: text}
  analyze:
    provider: modules.rca.AnalyzeFunc
  calibrate:
    provider: modules.rca.CalibrateRunner
  consume:
    circuit: circuits/asterisk-ingest.yaml

serve:
  provider: modules.rca.ServeConfig

demo:
  provider: modules.rca.DemoConfig
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "asterisk" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Imports) != 3 {
		t.Errorf("imports = %d, want 3", len(m.Imports))
	}
	if len(m.Embed) != 3 {
		t.Errorf("embed = %d, want 3", len(m.Embed))
	}
	if m.CLI.Analyze == nil || m.CLI.Analyze.Provider != "modules.rca.AnalyzeFunc" {
		t.Error("CLI.Analyze not parsed correctly")
	}
	if m.CLI.Calibrate == nil || m.CLI.Calibrate.Provider != "modules.rca.CalibrateRunner" {
		t.Error("CLI.Calibrate not parsed correctly")
	}
	if m.CLI.Consume == nil || m.CLI.Consume.Circuit != "circuits/asterisk-ingest.yaml" {
		t.Error("CLI.Consume not parsed correctly")
	}
	if m.Serve == nil || m.Serve.Provider != "modules.rca.ServeConfig" {
		t.Error("Serve not parsed correctly")
	}
	if m.Demo == nil || m.Demo.Provider != "modules.rca.DemoConfig" {
		t.Error("Demo not parsed correctly")
	}
}

func TestParseManifest_MissingName(t *testing.T) {
	data := []byte(`description: no name`)
	_, err := ParseManifest(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}
