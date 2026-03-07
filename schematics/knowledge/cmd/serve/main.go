// Command serve runs the knowledge schematic as an MCP server over
// Streamable HTTP. It exposes knowledge.Reader operations as MCP tools
// (ensure, search, read, list) for consumption by other schematics.
//
// Usage: serve [--port=9100] [--driver=git,docs]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dpopsuev/origami/connectors/docs"
	"github.com/dpopsuev/origami/connectors/github"
	skn "github.com/dpopsuev/origami/schematics/knowledge"
)

func main() {
	port := flag.Int("port", 9100, "HTTP port for the MCP server")
	drivers := flag.String("drivers", "", "comma-separated driver names to register (e.g. git,docs)")
	flag.Parse()

	var opts []skn.RouterOption
	if *drivers != "" {
		for _, name := range strings.Split(*drivers, ",") {
			name = strings.TrimSpace(name)
			switch name {
			case "git":
				d, err := github.DefaultGitDriver()
				if err != nil {
					log.Fatalf("create git driver: %v", err)
				}
				opts = append(opts, skn.WithGitDriver(d))
				log.Printf("registered driver: git")
			case "docs":
				d, err := docs.DefaultDocsDriver()
				if err != nil {
					log.Fatalf("create docs driver: %v", err)
				}
				opts = append(opts, skn.WithDocsDriver(d))
				log.Printf("registered driver: docs")
			default:
				log.Fatalf("unknown driver %q (known: git, docs)", name)
			}
		}
	}
	router := skn.NewRouter(opts...)

	server := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "origami-knowledge", Version: "v0.1.0"},
		nil,
	)

	skn.RegisterTools(server, router)
	skn.RegisterSynthesizeTool(server, skn.SynthesizeToolOpts{
		Synthesizer: &skn.StructuralSynthesizer{},
		Router:      router,
	})

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
		if router.Ready() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

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

