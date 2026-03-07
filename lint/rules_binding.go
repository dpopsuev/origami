package lint

import (
	"github.com/dpopsuev/origami/fold"
)

// ManifestLintContext holds data needed for manifest-level lint rules.
type ManifestLintContext struct {
	Manifest *fold.Manifest
	File     string
}

// NewManifestLintContext creates a ManifestLintContext from a parsed manifest.
func NewManifestLintContext(m *fold.Manifest, file string) *ManifestLintContext {
	return &ManifestLintContext{
		Manifest: m,
		File:     file,
	}
}

// ManifestRule is a lint rule that operates on manifest-level data.
type ManifestRule interface {
	ID() string
	Description() string
	Severity() Severity
	Tags() []string
	CheckManifest(ctx *ManifestLintContext) []Finding
}

// RunManifestRules runs all manifest-level rules and returns findings.
func RunManifestRules(ctx *ManifestLintContext) []Finding {
	var rules []ManifestRule
	var findings []Finding
	for _, rule := range rules {
		findings = append(findings, rule.CheckManifest(ctx)...)
	}
	return findings
}
