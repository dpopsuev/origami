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

type serveBackendFlags []string

func (b *serveBackendFlags) String() string { return strings.Join(*b, ", ") }
func (b *serveBackendFlags) Set(val string) error {
	*b = append(*b, val)
	return nil
}

func serveCmd(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := fs.Int("port", 9000, "HTTP port for the gateway")
	var backends serveBackendFlags
	fs.Var(&backends, "backend", "Backend in name=url format (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(backends) == 0 {
		return fmt.Errorf("at least one --backend is required\nusage: origami serve --port 9000 --backend rca=http://localhost:9200/mcp")
	}

	var configs []gateway.BackendConfig
	for _, b := range backends {
		parts := strings.SplitN(b, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid backend format %q, expected name=url", b)
		}
		configs = append(configs, gateway.BackendConfig{Name: parts[0], Endpoint: parts[1]})
	}

	gw := gateway.New(configs)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := gw.Start(ctx); err != nil {
		return fmt.Errorf("start gateway: %w", err)
	}
	defer gw.Stop(context.Background())

	addr := fmt.Sprintf(":%d", *port)
	httpServer := &http.Server{Addr: addr, Handler: gw.Handler()}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("origami gateway listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
