package fold

import (
	"strings"
	"testing"
)

func TestGenerateDomainServe(t *testing.T) {
	m := &Manifest{
		Name:    "asterisk",
		Version: "1.0",
		DomainServe: &DomainServeConfig{
			Port:  9300,
			Embed: "internal/",
		},
	}

	src, err := GenerateDomainServe(m)
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)

	for _, want := range []string{
		"DO NOT EDIT",
		"package main",
		"domainserve.New(",
		`"github.com/dpopsuev/origami/domainserve"`,
		"//go:embed internal",
		"var domainData embed.FS",
		`"asterisk"`,
		`"1.0"`,
		"9300",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("missing %q in generated code:\n%s", want, code)
		}
	}
}

func TestGenerateDomainServe_DefaultPort(t *testing.T) {
	m := &Manifest{
		Name:    "myapp",
		Version: "2.0",
		DomainServe: &DomainServeConfig{
			Embed: "data/",
		},
	}

	src, err := GenerateDomainServe(m)
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)

	if !strings.Contains(code, "9300") {
		t.Errorf("expected default port 9300 in:\n%s", code)
	}
}

func TestGenerateDomainServe_NilConfig(t *testing.T) {
	m := &Manifest{Name: "test"}
	_, err := GenerateDomainServe(m)
	if err == nil {
		t.Fatal("expected error for nil domain_serve config")
	}
	if !strings.Contains(err.Error(), "domain_serve") {
		t.Errorf("error should mention domain_serve, got: %v", err)
	}
}

func TestGenerateDomainServe_MissingEmbed(t *testing.T) {
	m := &Manifest{
		Name:        "test",
		DomainServe: &DomainServeConfig{Port: 9300},
	}
	_, err := GenerateDomainServe(m)
	if err == nil {
		t.Fatal("expected error for missing embed directory")
	}
	if !strings.Contains(err.Error(), "embed") {
		t.Errorf("error should mention embed, got: %v", err)
	}
}
