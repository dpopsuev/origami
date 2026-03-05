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
	HasSubprocess   bool
	Bindings        []bindingEntry
	Secondaries     []secondaryEntry
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

// secondaryEntry represents a secondary schematic that must be constructed
// before the primary schematic's Apply() call.
type secondaryEntry struct {
	VarName   string         // Go variable name for the constructed schematic
	Alias     string         // import alias for the secondary schematic package
	Factory   string         // factory function name (e.g. "NewRouter")
	OptionCmd string         // primary schematic's option func (e.g. "WithKnowledgeReader")
	Bindings  []bindingEntry // bindings for the secondary schematic's sockets
	Mode      string         // "in-process", "subprocess", or "container"
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

		alias := safeAlias(goPath)

		if ctx.CmdImport == "" {
			cmdPath := goPath + "/cmd"
			ctx.CmdImport = cmdPath
			ctx.CmdAlias = alias + "cmd"
			seen[goPath] = true
			seen[cmdPath] = true
		}
		// Non-primary schematics are NOT marked seen here — they'll be added
		// to Imports by resolveSecondaries when their schematic socket is resolved.
	}

	if ctx.CmdImport == "" {
		return nil, fmt.Errorf("manifest must import at least one module with a cmd/ package")
	}

	// Partition bindings: schematic sockets go to secondary construction,
	// connector sockets go to normal binding resolution.
	schematicSockets := buildSchematicSocketMap(reg, ctx)
	socketBindings := make(map[string]string, len(m.Bindings))
	secondaryBindings := map[string]map[string]string{} // namespace -> socket -> connector

	for k, v := range m.Bindings {
		// Check raw key for secondary schematic bindings BEFORE namespace stripping.
		// "knowledge.git" -> namespace "knowledge", socket "git"
		parts := strings.SplitN(k, ".", 2)
		if len(parts) == 2 {
			ns := parts[0]
			if _, isSchematic := schematicSockets[ns]; isSchematic {
				if secondaryBindings[ns] == nil {
					secondaryBindings[ns] = map[string]string{}
				}
				secondaryBindings[ns][parts[1]] = v
				continue
			}
		}

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

	// Resolve secondary schematics
	if len(schematicSockets) > 0 {
		secondaries, err := resolveSecondaries(schematicSockets, secondaryBindings, m, reg, seen, ctx)
		if err != nil {
			return nil, err
		}
		ctx.Secondaries = secondaries
		for _, s := range secondaries {
			if s.Mode == "subprocess" || s.Mode == "container" {
				ctx.HasSubprocess = true
				break
			}
		}
	}

	if len(m.Sources) > 0 {
		ctx.SourcePacks = m.Sources
	}

	return ctx, nil
}

// schematicSocketInfo holds info about a socket that is satisfied by a secondary schematic.
type schematicSocketInfo struct {
	SocketName    string // socket name on the primary schematic (e.g. "knowledge")
	OptionFunc    string // primary's With* option (e.g. "WithKnowledgeReader")
	ComponentName string // secondary schematic's component name (e.g. "origami-knowledge")
}

// buildSchematicSocketMap identifies sockets on the primary schematic that
// have a `schematic:` field set (satisfied by constructing another schematic).
// Returns namespace -> schematicSocketInfo.
func buildSchematicSocketMap(reg ModuleRegistry, ctx *templateContext) map[string]schematicSocketInfo {
	result := map[string]schematicSocketInfo{}

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
		if sock.Schematic != "" {
			result[sock.Name] = schematicSocketInfo{
				SocketName:    sock.Name,
				OptionFunc:    sock.Option,
				ComponentName: sock.Schematic,
			}
		}
	}
	return result
}

// resolveSecondaries constructs secondaryEntry values for each schematic-typed socket.
func resolveSecondaries(
	sockets map[string]schematicSocketInfo,
	bindings map[string]map[string]string,
	m *Manifest,
	reg ModuleRegistry,
	seen map[string]bool,
	ctx *templateContext,
) ([]secondaryEntry, error) {
	// Build component name -> import FQCN index from manifest imports
	componentIndex := map[string]string{} // component name -> go path
	for _, imp := range m.Imports {
		goPath, err := reg.ResolveFQCN(imp)
		if err != nil {
			continue
		}
		meta, err := loadComponentMetaForModule(goPath)
		if err != nil {
			continue
		}
		componentIndex[meta.Component] = goPath
	}

	var entries []secondaryEntry
	socketNames := make([]string, 0, len(sockets))
	for name := range sockets {
		socketNames = append(socketNames, name)
	}
	sort.Strings(socketNames)

	for _, name := range socketNames {
		info := sockets[name]
		goPath, ok := componentIndex[info.ComponentName]
		if !ok {
			// Secondary schematic not imported — skip (optional dependency)
			continue
		}

		meta, err := loadComponentMetaForModule(goPath)
		if err != nil {
			return nil, fmt.Errorf("load component.yaml for %q: %w", info.ComponentName, err)
		}

		alias := safeAlias(goPath)
		if !seen[goPath] {
			seen[goPath] = true
			ctx.Imports = append(ctx.Imports, importEntry{Alias: alias, Path: goPath})
		}

		// Determine deploy mode
		mode := "in-process"
		if m.Deploy != nil {
			if dc := m.Deploy[name]; dc != nil && dc.Mode != "" {
				mode = dc.Mode
			}
		}

		// Resolve the secondary's own socket bindings
		var secBindings []bindingEntry
		if sb, ok := bindings[name]; ok && len(sb) > 0 {
			secSocketMap := map[string]string{}
			for _, sock := range meta.Requires.Sockets {
				if sock.Option != "" {
					secSocketMap[sock.Name] = sock.Option
				}
			}

			secSocketNames := make([]string, 0, len(sb))
			for s := range sb {
				secSocketNames = append(secSocketNames, s)
			}
			sort.Strings(secSocketNames)

			for _, sockName := range secSocketNames {
				connFQCN := sb[sockName]
				connGoPath, err := reg.ResolveFQCN(connFQCN)
				if err != nil {
					return nil, fmt.Errorf("resolve secondary binding %s.%s -> %q: %w", name, sockName, connFQCN, err)
				}

				connAlias := safeAlias(connGoPath)
				if !seen[connGoPath] {
					seen[connGoPath] = true
					ctx.Imports = append(ctx.Imports, importEntry{Alias: connAlias, Path: connGoPath})
				}

				optFunc, ok := secSocketMap[sockName]
				if !ok {
					return nil, fmt.Errorf("unknown socket %q on secondary schematic %q", sockName, info.ComponentName)
				}

				factory := lookupConnectorFactory(sockName, connGoPath)

				secBindings = append(secBindings, bindingEntry{
					Socket:      sockName,
					OptionFunc:  optFunc,
					FactoryFunc: factory,
					ConnAlias:   connAlias,
				})
			}
		}

		entries = append(entries, secondaryEntry{
			VarName:   name + "Schematic",
			Alias:     alias,
			Factory:   meta.Factory,
			OptionCmd: info.OptionFunc,
			Bindings:  secBindings,
			Mode:      mode,
		})
	}
	return entries, nil
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
{{- if .HasSubprocess }}
	"context"
	"log"

	"github.com/dpopsuev/origami/subprocess"
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
{{- range .Secondaries }}
{{- $sec := . }}
{{- if eq .Mode "in-process" }}
	{{ .VarName }} := {{ .Alias }}.{{ .Factory }}(
{{- range .Bindings }}
		{{ $sec.Alias }}.{{ .OptionFunc }}({{ .ConnAlias }}.{{ .FactoryFunc }}),
{{- end }}
	)
{{- else if eq .Mode "subprocess" }}
	{{ .VarName }}Srv := &subprocess.Server{BinaryPath: "{{ .VarName }}"}
	if err := {{ .VarName }}Srv.Start(context.Background()); err != nil {
		log.Fatalf("start {{ .VarName }}: %v", err)
	}
	defer {{ .VarName }}Srv.Stop(context.Background())
{{- end }}
{{- end }}
{{- if or .Bindings .StoreSchemaFile .SourcePacks .Secondaries }}
	{{ .CmdAlias }}.Apply(
{{- if .StoreSchemaFile }}
		{{ .CmdAlias }}.WithStoreSchema(storeSchema),
{{- end }}
{{- range .Secondaries }}
{{- if eq .Mode "in-process" }}
		{{ $.CmdAlias }}.{{ .OptionCmd }}({{ .VarName }}),
{{- else if eq .Mode "subprocess" }}
		{{ $.CmdAlias }}.{{ .OptionCmd }}({{ .VarName }}Srv),
{{- end }}
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
