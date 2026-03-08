// Command rca-serve runs the RCA circuit as a standalone MCP server.
// It connects to a domain data server via MCPRemoteFS and optionally
// to a knowledge server for code access during investigation.
//
// Usage: rca-serve [--port=9200] --domain-endpoint http://domain:9300/mcp [--knowledge-endpoint http://knowledge:9100/mcp]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/domainfs"
	mcpserver "github.com/dpopsuev/origami/schematics/rca/mcpconfig"
	"github.com/dpopsuev/origami/subprocess"
)

type sessionToolCaller struct {
	session *sdkmcp.ClientSession
}

func (s *sessionToolCaller) CallTool(ctx context.Context, name string, args map[string]any) (*sdkmcp.CallToolResult, error) {
	return s.session.CallTool(ctx, &sdkmcp.CallToolParams{Name: name, Arguments: args})
}

var _ subprocess.ToolCaller = (*sessionToolCaller)(nil)

func connectMCP(ctx context.Context, endpoint, label string) *sdkmcp.ClientSession {
	transport := &sdkmcp.StreamableClientTransport{Endpoint: endpoint}
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "origami-rca-engine", Version: "v0.1.0"},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("connect to %s at %s: %v", label, endpoint, err)
	}
	return session
}

func main() {
	port := flag.Int("port", 9200, "HTTP port for the RCA MCP server")
	domainEndpoint := flag.String("domain-endpoint", envOr("DOMAIN_ENDPOINT", ""), "Domain data MCP endpoint (required)")
	knowledgeEndpoint := flag.String("knowledge-endpoint", envOr("KNOWLEDGE_ENDPOINT", ""), "Knowledge MCP endpoint (optional)")
	productName := flag.String("product", envOr("PRODUCT_NAME", "asterisk"), "Product name for state directory")
	flag.Parse()

	if *domainEndpoint == "" {
		log.Fatal("--domain-endpoint is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	domainSession := connectMCP(ctx, *domainEndpoint, "domain")
	defer domainSession.Close()
	remoteFS := domainfs.New(&sessionToolCaller{session: domainSession}).
		WithTimeout(10 * time.Second)
	log.Printf("connected to domain server at %s", *domainEndpoint)

	opts := []mcpserver.ServerOption{
		mcpserver.WithDomainFS(remoteFS),
	}

	if *knowledgeEndpoint != "" {
		knSession := connectMCP(ctx, *knowledgeEndpoint, "knowledge")
		defer knSession.Close()
		log.Printf("connected to knowledge server at %s", *knowledgeEndpoint)
	}

	srv := mcpserver.NewServer(*productName, opts...)
	defer srv.Shutdown()

	mcpHandler := sdkmcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *sdkmcp.Server { return srv.MCPServer },
		&sdkmcp.StreamableHTTPOptions{Stateless: false},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if srv.MCPServer != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("rca engine listening on %s (domain: %s)", addr, *domainEndpoint)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
