package fold

import (
	"fmt"
	"regexp"
	"strings"
)

var validImportRE = regexp.MustCompile(`^[a-z][a-z0-9._/-]*$`)

// ModuleRegistry maps module prefixes to Go module paths.
// Default: "origami" → "github.com/dpopsuev/origami".
type ModuleRegistry map[string]string

// DefaultRegistry returns the standard module prefix mappings.
func DefaultRegistry() ModuleRegistry {
	return ModuleRegistry{
		"origami": "github.com/dpopsuev/origami",
	}
}

// ResolveFQCN converts a dot-separated FQCN to a Go import path.
// "origami.schematics.rca" → "github.com/dpopsuev/origami/schematics/rca"
func (r ModuleRegistry) ResolveFQCN(fqcn string) (string, error) {
	if !validImportRE.MatchString(fqcn) {
		return "", fmt.Errorf("invalid FQCN %q: must match %s", fqcn, validImportRE.String())
	}

	parts := strings.SplitN(fqcn, ".", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid FQCN %q: must have at least prefix.path", fqcn)
	}

	prefix := parts[0]
	modPath, ok := r[prefix]
	if !ok {
		return "", fmt.Errorf("unknown module prefix %q in FQCN %q (known: %s)", prefix, fqcn, r.knownPrefixes())
	}

	subPath := strings.ReplaceAll(parts[1], ".", "/")
	return modPath + "/" + subPath, nil
}

// ResolveProvider converts a provider FQCN into an import path and exported symbol.
// "schematics.rca.CalibrateRunner" → import "github.com/dpopsuev/origami/schematics/rca", symbol "CalibrateRunner"
// Provider FQCNs are implicitly prefixed with the "origami" module.
func (r ModuleRegistry) ResolveProvider(provider string) (importPath, symbol string, err error) {
	parts := strings.Split(provider, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("provider %q must have at least package.Symbol", provider)
	}

	symbol = parts[len(parts)-1]
	pkgFQCN := "origami." + strings.Join(parts[:len(parts)-1], ".")

	importPath, err = r.ResolveFQCN(pkgFQCN)
	if err != nil {
		return "", "", fmt.Errorf("resolve provider %q: %w", provider, err)
	}

	return importPath, symbol, nil
}

func (r ModuleRegistry) knownPrefixes() string {
	var prefixes []string
	for k := range r {
		prefixes = append(prefixes, k)
	}
	return strings.Join(prefixes, ", ")
}
