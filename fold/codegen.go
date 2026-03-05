package fold

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"text/template"
)

// GenerateMain produces Go source for a main.go that builds the CLI
// from the manifest. The generated code imports the module cmd package
// and calls its Execute() function.
func GenerateMain(m *Manifest, reg ModuleRegistry) ([]byte, error) {
	ctx, err := buildTemplateContext(m, reg)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

type templateContext struct {
	Name            string
	Imports         []importEntry
	CmdImport       string
	CmdAlias        string
	HasVersion      bool
	Bindings        []bindingEntry
	StoreSchemaFile string
	SourcePacks     map[string]string
}

type importEntry struct {
	Alias string
	Path  string
}

type bindingEntry struct {
	Socket      string
	OptionFunc  string
	FactoryFunc string
	ConnAlias   string
}

func buildTemplateContext(m *Manifest, reg ModuleRegistry) (*templateContext, error) {
	ctx := &templateContext{
		Name:       m.Name,
		HasVersion: m.Version != "",
	}

	seen := map[string]bool{}

	for _, imp := range m.Imports {
		goPath, err := reg.ResolveFQCN(imp)
		if err != nil {
			return nil, fmt.Errorf("resolve import %q: %w", imp, err)
		}
		if seen[goPath] {
			continue
		}
		seen[goPath] = true

		alias := safeAlias(goPath)
		cmdPath := goPath + "/cmd"

		if !seen[cmdPath] {
			ctx.CmdImport = cmdPath
			ctx.CmdAlias = alias + "cmd"
			seen[cmdPath] = true
		}
	}

	if ctx.CmdImport == "" {
		return nil, fmt.Errorf("manifest must import at least one module with a cmd/ package")
	}

	socketBindings := make(map[string]string, len(m.Bindings))
	for k, v := range m.Bindings {
		socketName := stripNamespace(k)
		if socketName == "store.schema" {
			ctx.StoreSchemaFile = path.Base(v)
			continue
		}
		socketBindings[socketName] = v
	}

	if len(socketBindings) > 0 {
		bindings, err := resolveBindings(socketBindings, reg, seen, ctx)
		if err != nil {
			return nil, err
		}
		ctx.Bindings = bindings
	}

	if len(m.Sources) > 0 {
		ctx.SourcePacks = m.Sources
	}

	return ctx, nil
}

func resolveBindings(bindings map[string]string, reg ModuleRegistry, seen map[string]bool, ctx *templateContext) ([]bindingEntry, error) {
	schematicMeta := buildSocketOptionMap(reg, ctx)

	sockets := make([]string, 0, len(bindings))
	for s := range bindings {
		sockets = append(sockets, s)
	}
	sort.Strings(sockets)

	var entries []bindingEntry
	for _, socket := range sockets {
		connFQCN := bindings[socket]
		goPath, err := reg.ResolveFQCN(connFQCN)
		if err != nil {
			return nil, fmt.Errorf("resolve binding %q -> %q: %w", socket, connFQCN, err)
		}

		alias := safeAlias(goPath)
		if !seen[goPath] {
			seen[goPath] = true
			ctx.Imports = append(ctx.Imports, importEntry{Alias: alias, Path: goPath})
		}

		optFunc, ok := schematicMeta[socket]
		if !ok {
			known := make([]string, 0, len(schematicMeta))
			for k := range schematicMeta {
				known = append(known, k)
			}
			sort.Strings(known)
			return nil, fmt.Errorf("unknown socket %q in bindings (known: %s)", socket, strings.Join(known, ", "))
		}

		factory := lookupConnectorFactory(socket, goPath)

		entries = append(entries, bindingEntry{
			Socket:      socket,
			OptionFunc:  optFunc,
			FactoryFunc: factory,
			ConnAlias:   alias,
		})
	}
	return entries, nil
}

// buildSocketOptionMap reads component.yaml from each imported schematic
// and builds a socket->option function map. Falls back to hardcoded defaults
// if component.yaml cannot be loaded.
func buildSocketOptionMap(reg ModuleRegistry, ctx *templateContext) map[string]string {
	result := map[string]string{}

	cmdImport := ctx.CmdImport
	if cmdImport == "" {
		return result
	}
	schematicPath := strings.TrimSuffix(cmdImport, "/cmd")

	meta, err := loadComponentMetaForModule(schematicPath)
	if err != nil {
		return result
	}

	for _, sock := range meta.Requires.Sockets {
		if sock.Option != "" {
			result[sock.Name] = sock.Option
		}
	}
	return result
}

// lookupConnectorFactory reads component.yaml from the connector module
// and returns the factory function for the given socket. Falls back to "New".
func lookupConnectorFactory(socket, goPath string) string {
	meta, err := loadComponentMetaForModule(goPath)
	if err != nil {
		return "New"
	}

	for _, sat := range meta.Satisfies {
		if sat.Socket == socket {
			return sat.Factory
		}
	}
	return "New"
}

// stripNamespace removes the namespace prefix from a binding key.
// "rca.source" -> "source", "store.schema" -> "store.schema" (no namespace),
// "rca.store.schema" -> "store.schema".
//
// Heuristic: a key is namespaced when the first dot-segment is NOT a
// known compound-socket prefix (like "store"). Compound sockets use
// dots internally (e.g. "store.schema") and must not be split.
var compoundSocketPrefixes = map[string]bool{
	"store": true,
}

func stripNamespace(key string) string {
	dot := strings.IndexByte(key, '.')
	if dot < 0 {
		return key
	}
	firstSeg := key[:dot]
	if compoundSocketPrefixes[firstSeg] {
		return key
	}
	return key[dot+1:]
}

func safeAlias(importPath string) string {
	base := path.Base(importPath)
	base = strings.ReplaceAll(base, "-", "")
	return base
}

var mainTemplate = `// Code generated by origami fold. DO NOT EDIT.
package main

import (
{{- if .StoreSchemaFile }}
	_ "embed"
{{- end }}
	{{ .CmdAlias }} "{{ .CmdImport }}"
{{- range .Imports }}
	{{ .Alias }} "{{ .Path }}"
{{- end }}
)
{{- if .StoreSchemaFile }}

//go:embed {{ .StoreSchemaFile }}
var storeSchema []byte
{{- end }}

func main() {
{{- if or .Bindings .StoreSchemaFile .SourcePacks }}
	{{ .CmdAlias }}.Apply(
{{- if .StoreSchemaFile }}
		{{ .CmdAlias }}.WithStoreSchema(storeSchema),
{{- end }}
{{- range .Bindings }}
		{{ $.CmdAlias }}.{{ .OptionFunc }}({{ .ConnAlias }}.{{ .FactoryFunc }}),
{{- end }}
{{- if .SourcePacks }}
		{{ .CmdAlias }}.WithSourcePacks(map[string]string{
{{- range $name, $path := .SourcePacks }}
			{{ printf "%q" $name }}: {{ printf "%q" $path }},
{{- end }}
		}),
{{- end }}
	)
{{- end }}
	{{ .CmdAlias }}.Execute()
}
`
