package lint

import (
	"fmt"
	"os"
	"path/filepath"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/fold"
)

// ManifestLintContext holds data needed for manifest-level lint rules.
type ManifestLintContext struct {
	Manifest   *fold.Manifest
	Registry   fold.ModuleRegistry
	File       string
	Components map[string]*framework.ComponentManifest
}

// NewManifestLintContext creates a ManifestLintContext from a parsed manifest.
// It resolves component.yaml files from imported and bound FQCNs.
func NewManifestLintContext(m *fold.Manifest, reg fold.ModuleRegistry, file string) *ManifestLintContext {
	ctx := &ManifestLintContext{
		Manifest:   m,
		Registry:   reg,
		File:       file,
		Components: make(map[string]*framework.ComponentManifest),
	}
	ctx.loadComponents()
	return ctx
}

func (c *ManifestLintContext) loadComponents() {
	fqcns := make(map[string]bool)
	for _, imp := range c.Manifest.Imports {
		fqcns[imp] = true
	}
	for _, connFQCN := range c.Manifest.Bindings {
		fqcns[connFQCN] = true
	}

	for fqcn := range fqcns {
		goPath, err := c.Registry.ResolveFQCN(fqcn)
		if err != nil {
			continue
		}
		comp := resolveComponentYAML(goPath)
		if comp != nil {
			c.Components[fqcn] = comp
		}
	}
}

func resolveComponentYAML(goPath string) *framework.ComponentManifest {
	home, _ := os.UserHomeDir()
	base := filepath.Base(goPath)

	parent := goPath
	for {
		parts := filepath.Dir(parent)
		if parts == "." || parts == "/" || parts == parent {
			break
		}
		parent = parts
	}

	modBase := filepath.Base(filepath.Dir(filepath.Dir(goPath)))
	if modBase == "." {
		modBase = base
	}

	subPath := ""
	parts := filepath.SplitList(goPath)
	_ = parts

	idx := 0
	for i, c := range goPath {
		if c == '/' {
			idx++
			if idx == 3 {
				subPath = goPath[i+1:]
				break
			}
		}
	}

	candidates := []string{
		filepath.Join(home, "Workspace", modBase, subPath, "component.yaml"),
	}

	for _, path := range candidates {
		comp, err := framework.LoadComponentManifest(path)
		if err == nil {
			return comp
		}
	}
	return nil
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
	rules := []ManifestRule{
		&UnboundSocket{},
		&UnknownBinding{},
		&UnsatisfiedConnector{},
		&MissingConnector{},
	}

	var findings []Finding
	for _, rule := range rules {
		findings = append(findings, rule.CheckManifest(ctx)...)
	}
	return findings
}

// M1: UnboundSocket — schematic declares a socket with no binding in the manifest.
type UnboundSocket struct{}

func (r *UnboundSocket) ID() string          { return "M1/unbound-socket" }
func (r *UnboundSocket) Description() string { return "schematic socket has no binding in manifest" }
func (r *UnboundSocket) Severity() Severity  { return SeverityError }
func (r *UnboundSocket) Tags() []string      { return []string{"binding", "manifest"} }

func (r *UnboundSocket) CheckManifest(ctx *ManifestLintContext) []Finding {
	var findings []Finding
	for fqcn, comp := range ctx.Components {
		for _, sock := range comp.Requires.Sockets {
			if _, bound := ctx.Manifest.Bindings[sock.Name]; !bound {
				findings = append(findings, Finding{
					RuleID:   r.ID(),
					Severity: r.Severity(),
					Message:  fmt.Sprintf("socket %q declared by %s has no binding in manifest", sock.Name, fqcn),
					File:     ctx.File,
				})
			}
		}
	}
	return findings
}

// M2: UnknownBinding — manifest binds a name the schematic doesn't declare.
type UnknownBinding struct{}

func (r *UnknownBinding) ID() string          { return "M2/unknown-binding" }
func (r *UnknownBinding) Description() string { return "manifest binds a name no schematic declares" }
func (r *UnknownBinding) Severity() Severity  { return SeverityError }
func (r *UnknownBinding) Tags() []string      { return []string{"binding", "manifest"} }

func (r *UnknownBinding) CheckManifest(ctx *ManifestLintContext) []Finding {
	declaredSockets := make(map[string]bool)
	for _, comp := range ctx.Components {
		for _, sock := range comp.Requires.Sockets {
			declaredSockets[sock.Name] = true
		}
	}

	knownCodegenSockets := map[string]bool{
		"source": true, "store": true,
		"pusher": true, "launch_fetcher": true,
	}

	var findings []Finding
	for name := range ctx.Manifest.Bindings {
		if !declaredSockets[name] && !knownCodegenSockets[name] {
			findings = append(findings, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("binding %q does not match any declared socket", name),
				File:     ctx.File,
			})
		}
	}
	return findings
}

// M3: UnsatisfiedConnector — connector doesn't declare `satisfies` for the bound socket.
type UnsatisfiedConnector struct{}

func (r *UnsatisfiedConnector) ID() string          { return "M3/unsatisfied-connector" }
func (r *UnsatisfiedConnector) Description() string { return "connector does not satisfy the bound socket" }
func (r *UnsatisfiedConnector) Severity() Severity  { return SeverityError }
func (r *UnsatisfiedConnector) Tags() []string      { return []string{"binding", "manifest"} }

func (r *UnsatisfiedConnector) CheckManifest(ctx *ManifestLintContext) []Finding {
	var findings []Finding
	for socket, connFQCN := range ctx.Manifest.Bindings {
		comp, ok := ctx.Components[connFQCN]
		if !ok {
			continue
		}
		satisfied := false
		for _, s := range comp.Satisfies {
			if s.Socket == socket {
				satisfied = true
				break
			}
		}
		if !satisfied {
			findings = append(findings, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("connector %s does not declare satisfies for socket %q", connFQCN, socket),
				File:     ctx.File,
			})
		}
	}
	return findings
}

// M4: MissingConnector — binding references an FQCN that cannot be resolved.
type MissingConnector struct{}

func (r *MissingConnector) ID() string          { return "M4/missing-connector" }
func (r *MissingConnector) Description() string { return "binding references unresolvable FQCN" }
func (r *MissingConnector) Severity() Severity  { return SeverityError }
func (r *MissingConnector) Tags() []string      { return []string{"binding", "manifest"} }

func (r *MissingConnector) CheckManifest(ctx *ManifestLintContext) []Finding {
	var findings []Finding
	for socket, connFQCN := range ctx.Manifest.Bindings {
		if _, err := ctx.Registry.ResolveFQCN(connFQCN); err != nil {
			findings = append(findings, Finding{
				RuleID:   r.ID(),
				Severity: r.Severity(),
				Message:  fmt.Sprintf("binding %q references unresolvable FQCN %q: %v", socket, connFQCN, err),
				File:     ctx.File,
			})
		}
	}
	return findings
}
