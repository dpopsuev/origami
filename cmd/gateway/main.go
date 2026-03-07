// Command gateway runs an MCP routing proxy that dispatches tool calls
// to backend MCP services based on tool name.
//
// Usage: gateway [--port=9000] --backend rca=http://localhost:9200/mcp --backend knowledge=http://localhost:9100/mcp
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

	"github.com/dpopsuev/origami/gateway"
)

type backendFlags []string

func (b *backendFlags) String() string { return strings.Join(*b, ", ") }
func (b *backendFlags) Set(val string) error {
	*b = append(*b, val)
	return nil
}

func main() {
	port := flag.Int("port", 9000, "HTTP port for the gateway")
	var backends backendFlags
	flag.Var(&backends, "backend", "Backend in name=url format (repeatable)")
	flag.Parse()

	if len(backends) == 0 {
		log.Fatal("at least one --backend is required")
	}

	var configs []gateway.BackendConfig
	for _, b := range backends {
		parts := strings.SplitN(b, "=", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid backend format %q, expected name=url", b)
		}
		configs = append(configs, gateway.BackendConfig{Name: parts[0], Endpoint: parts[1]})
	}

	gw := gateway.New(configs)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := gw.Start(ctx); err != nil {
		log.Fatalf("start gateway: %v", err)
	}
	defer gw.Stop(context.Background())

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{Addr: addr, Handler: gw.Handler()}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("gateway listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
