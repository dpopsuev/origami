// Package domainserve provides a reusable library for building domain
// data MCP servers. Any product calls domainserve.New(embedFS, config)
// and gets a ready-to-serve http.Handler with /mcp, /healthz, /readyz.
package domainserve

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// Config configures the domain data server.
type Config struct {
	Name    string
	Version string
}

// CircuitInfo describes one circuit found in the domain filesystem.
type CircuitInfo struct {
	Name        string `json:"name"`
	Topology    string `json:"topology,omitempty"`
	Description string `json:"description,omitempty"`
}

// DomainInfo is the response payload for the domain_info tool.
type DomainInfo struct {
	Name     string        `json:"name"`
	Version  string        `json:"version"`
	Circuits []CircuitInfo `json:"circuits"`
}

// DirEntry is a single entry returned by the domain_list tool.
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

// New creates an http.Handler that serves domain data from fsys over
// MCP. The handler exposes /mcp (Streamable HTTP), /healthz, /readyz.
func New(fsys fs.FS, cfg Config) http.Handler {
	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: cfg.Name, Version: cfg.Version},
		nil,
	)

	registerTools(server, fsys, cfg)

	mcpHandler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

func registerTools(server *sdkmcp.Server, fsys fs.FS, cfg Config) {
	server.AddTool(
		&sdkmcp.Tool{
			Name:        "domain_info",
			Description: "Return domain metadata including available circuits",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		func(_ context.Context, _ *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			info := DomainInfo{
				Name:     cfg.Name,
				Version:  cfg.Version,
				Circuits: scanCircuits(fsys),
			}
			data, err := json.Marshal(info)
			if err != nil {
				return errResult("marshal domain info: " + err.Error()), nil
			}
			return textResult(string(data)), nil
		},
	)

	server.AddTool(
		&sdkmcp.Tool{
			Name:        "domain_read",
			Description: "Read a file from the domain filesystem",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path to read"}},"required":["path"]}`),
		},
		func(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			if !fs.ValidPath(args.Path) {
				return errResult("invalid path: " + args.Path), nil
			}
			data, err := fs.ReadFile(fsys, args.Path)
			if err != nil {
				return errResult(err.Error()), nil
			}
			return textResult(string(data)), nil
		},
	)

	server.AddTool(
		&sdkmcp.Tool{
			Name:        "domain_list",
			Description: "List entries in a domain filesystem directory",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path to list"}},"required":["path"]}`),
		},
		func(_ context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			if !fs.ValidPath(args.Path) {
				return errResult("invalid path: " + args.Path), nil
			}
			entries, err := fs.ReadDir(fsys, args.Path)
			if err != nil {
				return errResult(err.Error()), nil
			}
			result := make([]DirEntry, len(entries))
			for i, e := range entries {
				result[i] = DirEntry{Name: e.Name(), IsDir: e.IsDir()}
			}
			data, err := json.Marshal(result)
			if err != nil {
				return errResult("marshal entries: " + err.Error()), nil
			}
			return textResult(string(data)), nil
		},
	)
}

// scanCircuits reads circuits/ directory and extracts metadata from
// each YAML file. Returns nil (not error) when circuits/ doesn't exist.
func scanCircuits(fsys fs.FS) []CircuitInfo {
	entries, err := fs.ReadDir(fsys, "circuits")
	if err != nil {
		return nil
	}
	var circuits []CircuitInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		ci := CircuitInfo{Name: name}

		data, err := fs.ReadFile(fsys, "circuits/"+e.Name())
		if err == nil {
			var header struct {
				Topology    string `yaml:"topology"`
				Description string `yaml:"description"`
			}
			if yaml.Unmarshal(data, &header) == nil {
				ci.Topology = header.Topology
				ci.Description = header.Description
			}
		}
		circuits = append(circuits, ci)
	}
	return circuits
}

func textResult(text string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
	}
}

func errResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: msg}},
	}
}
