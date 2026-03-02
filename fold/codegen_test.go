package fold

import (
	"strings"
	"testing"
)

func TestGenerateMain(t *testing.T) {
	m := &Manifest{
		Name:        "asterisk",
		Description: "Evidence-based RCA",
		Version:     "1.0",
		Imports:     []string{"origami.modules.rca"},
	}

	src, err := GenerateMain(m, DefaultRegistry())
	if err != nil {
		t.Fatal(err)
	}

	code := string(src)

	if !strings.Contains(code, "DO NOT EDIT") {
		t.Error("missing DO NOT EDIT comment")
	}
	if !strings.Contains(code, `"github.com/dpopsuev/origami/modules/rca/cmd"`) {
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
