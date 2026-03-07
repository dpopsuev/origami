package fold

import (
	"testing"
)

func TestParseManifest_Minimal(t *testing.T) {
	data := []byte(`
name: test-tool
description: A test tool
version: "1.0"
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
}

func TestParseManifest_DomainServe(t *testing.T) {
	data := []byte(`
name: asterisk
version: "1.0"
domain_serve:
  port: 9300
  embed: internal/
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.DomainServe == nil {
		t.Fatal("domain_serve is nil")
	}
	if m.DomainServe.Port != 9300 {
		t.Errorf("port = %d, want 9300", m.DomainServe.Port)
	}
	if m.DomainServe.Embed != "internal/" {
		t.Errorf("embed = %q, want internal/", m.DomainServe.Embed)
	}
}

func TestParseManifest_MissingName(t *testing.T) {
	data := []byte(`description: no name`)
	_, err := ParseManifest(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}
