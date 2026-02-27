package backend

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"
)

// StudioServer serves the Visual Editor: embedded SPA + REST API.
type StudioServer struct {
	store  *EventStore
	api    *API
	port   int
	server *http.Server
}

// NewStudioServer creates a server on the given port.
// Pass the embedded filesystem from the CLI package that builds the SPA.
func NewStudioServer(port int, staticFS fs.FS) *StudioServer {
	store := NewEventStore()
	api := NewAPI(store)

	mux := http.NewServeMux()

	// API routes take precedence
	mux.Handle("/api/", api.Handler())

	// Static SPA files (fallback to index.html for SPA routing)
	if staticFS != nil {
		fileServer := http.FileServerFS(staticFS)
		mux.Handle("/", spaHandler(fileServer, staticFS))
	}

	return &StudioServer{
		store: store,
		api:   api,
		port:  port,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 0, // SSE needs no write timeout
		},
	}
}

// Store returns the event store for observer registration.
func (s *StudioServer) Store() *EventStore { return s.store }

// Start begins serving in a background goroutine.
func (s *StudioServer) Start() {
	go func() {
		log.Printf("studio: listening on http://localhost:%d", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("studio: server error: %v", err)
		}
	}()
}

// Stop gracefully shuts down the server.
func (s *StudioServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// spaHandler serves static files and falls back to index.html for SPA routes.
func spaHandler(fileServer http.Handler, staticFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if path[0] == '/' {
			path = path[1:]
		}

		if _, err := fs.Stat(staticFS, path); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// PlaceholderFS is an embedded filesystem for development when the SPA hasn't been built.
//
//go:embed placeholder.html
var placeholderContent embed.FS

// PlaceholderStaticFS returns a filesystem with a minimal placeholder page.
func PlaceholderStaticFS() fs.FS {
	sub, _ := fs.Sub(placeholderContent, ".")
	return sub
}
