// Command serve runs the knowledge schematic as an MCP server over
// Streamable HTTP. It exposes knowledge.Reader operations as MCP tools
// (ensure, search, read, list) for consumption by other schematics.
//
// Usage: serve [--port=9100]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	skn "github.com/dpopsuev/origami/schematics/knowledge"
)

func main() {
	port := flag.Int("port", 9100, "HTTP port for the MCP server")
	flag.Parse()

	router := skn.NewRouter()

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "origami-knowledge", Version: "v0.1.0"},
		nil,
	)

	registerTools(server, router)

	handler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{Stateless: true},
	)

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{Addr: addr, Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("knowledge schematic listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func registerTools(server *sdkmcp.Server, router *skn.AccessRouter) {
	server.AddTool(
		&sdkmcp.Tool{
			Name:        "knowledge_ensure",
			Description: "Ensure a knowledge source is available (e.g. clone a repo)",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"object","description":"Knowledge source descriptor"}}}`),
		},
		func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Source skn.Source `json:"source"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			if err := router.Ensure(ctx, args.Source); err != nil {
				return errResult(err.Error()), nil
			}
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: "ok"}},
			}, nil
		},
	)

	server.AddTool(
		&sdkmcp.Tool{
			Name:        "knowledge_search",
			Description: "Search a knowledge source for matching content",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"object","description":"Knowledge source descriptor"},"query":{"type":"string","description":"Search query"},"max_results":{"type":"integer","description":"Maximum results to return"}}}`),
		},
		func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Source     skn.Source `json:"source"`
				Query      string    `json:"query"`
				MaxResults int       `json:"max_results"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			if args.MaxResults <= 0 {
				args.MaxResults = 10
			}
			results, err := router.Search(ctx, args.Source, args.Query, args.MaxResults)
			if err != nil {
				return errResult(err.Error()), nil
			}
			data, _ := json.Marshal(results)
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
			}, nil
		},
	)

	server.AddTool(
		&sdkmcp.Tool{
			Name:        "knowledge_read",
			Description: "Read content from a knowledge source at a given path",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"object","description":"Knowledge source descriptor"},"path":{"type":"string","description":"Path to read"}}}`),
		},
		func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Source skn.Source `json:"source"`
				Path   string    `json:"path"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			content, err := router.Read(ctx, args.Source, args.Path)
			if err != nil {
				return errResult(err.Error()), nil
			}
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(content)}},
			}, nil
		},
	)

	server.AddTool(
		&sdkmcp.Tool{
			Name:        "knowledge_list",
			Description: "List contents of a knowledge source directory",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"source":{"type":"object","description":"Knowledge source descriptor"},"root":{"type":"string","description":"Root path to list from"},"max_depth":{"type":"integer","description":"Maximum directory depth"}}}`),
		},
		func(ctx context.Context, req *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			var args struct {
				Source   skn.Source `json:"source"`
				Root     string    `json:"root"`
				MaxDepth int       `json:"max_depth"`
			}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return errResult("invalid arguments: " + err.Error()), nil
			}
			entries, err := router.List(ctx, args.Source, args.Root, args.MaxDepth)
			if err != nil {
				return errResult(err.Error()), nil
			}
			data, _ := json.Marshal(entries)
			return &sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
			}, nil
		},
	)
}

func errResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: msg}},
	}
}
