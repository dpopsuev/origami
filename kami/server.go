package kami

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	framework "github.com/dpopsuev/origami"
)

// Config controls the KamiServer behavior.
type Config struct {
	Port         int
	Bind         string // default "127.0.0.1"
	Debug        bool   // enable debug API endpoints
	Logger       *slog.Logger
	Bridge       *EventBridge
	SPA          http.FileSystem    // embedded frontend (nil = no SPA)
	Theme        Theme              // consumer theme (nil = default)
	Kabuki       KabukiConfig       // Kabuki presentation sections (nil = debugger-only mode)
	Vocab          framework.RichVocabulary   // rich vocabulary for node tooltips (nil = Theme only)
	MetricsHandler http.Handler               // Prometheus /metrics handler (nil = no metrics)
	Marbles        framework.MarbleRegistry  // marble registry for fold/unfold (nil = no marble API)
}

func (c *Config) addr() string {
	bind := c.Bind
	if bind == "" {
		bind = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", bind, c.Port)
}

func (c *Config) wsAddr() string {
	bind := c.Bind
	if bind == "" {
		bind = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", bind, c.Port+1)
}

func (c *Config) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return slog.Default()
}

// Server is the triple-homed Kami debugger process.
// It runs HTTP (SSE + SPA + browser events) and WS (AI commands to browser)
// on adjacent ports.
type Server struct {
	cfg    Config
	http   *http.Server
	ws     *http.Server
	bridge *EventBridge
	log    *slog.Logger

	mu      sync.Mutex
	wsConns map[int]*wsConn
	nextWS  int

	selMu     sync.RWMutex
	selection map[string]any
}

// NewServer creates a KamiServer. Call Start to begin serving.
func NewServer(cfg Config) *Server {
	s := &Server{
		cfg:     cfg,
		bridge:  cfg.Bridge,
		log:     cfg.logger(),
		wsConns: make(map[int]*wsConn),
	}
	return s
}

// Start begins serving HTTP and WS on the configured ports.
// Blocks until ctx is cancelled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/stream", s.handleSSE)
	mux.HandleFunc("POST /events/click", s.handleBrowserEvent("click"))
	mux.HandleFunc("POST /events/hover", s.handleBrowserEvent("hover"))
	mux.HandleFunc("POST /events/selection", s.handleBrowserEvent("selection"))
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/theme", s.handleThemeAPI)
	mux.HandleFunc("GET /api/pipeline", s.handlePipelineAPI)
	mux.HandleFunc("GET /api/kabuki", s.handleKabukiAPI)
	mux.HandleFunc("GET /api/marble/", s.handleMarbleAPI)

	if s.cfg.MetricsHandler != nil {
		mux.Handle("GET /metrics", s.cfg.MetricsHandler)
	}

	if s.cfg.SPA != nil {
		mux.Handle("GET /", http.FileServer(s.cfg.SPA))
	} else {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, "Kami debugger running. Frontend not embedded.")
		})
	}

	s.http = &http.Server{
		Addr:    s.cfg.addr(),
		Handler: mux,
	}

	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/", s.handleWS)
	s.ws = &http.Server{
		Addr:    s.cfg.wsAddr(),
		Handler: wsMux,
	}

	errCh := make(chan error, 2)

	go func() {
		s.log.Info("kami HTTP server starting", "addr", s.cfg.addr())
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP: %w", err)
		}
	}()

	go func() {
		s.log.Info("kami WS server starting", "addr", s.cfg.wsAddr())
		if err := s.ws.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("WS: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.http.Shutdown(shutCtx)
		s.ws.Shutdown(shutCtx)
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// handleSSE streams KamiEvents as Server-Sent Events.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	id, ch := s.bridge.Subscribe()
	defer s.bridge.Unsubscribe(id)

	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleBrowserEvent receives browser interaction events and emits them
// to the bridge so MCP tools can observe user interaction.
// Selection events are additionally stored for retrieval via GetSelection.
func (s *Server) handleBrowserEvent(eventType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if eventType == "selection" {
			s.SetSelection(payload)
		}
		s.bridge.Emit(Event{
			Type: EventType("browser_" + eventType),
			Data: payload,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetSelection returns the most recent browser selection payload, or nil.
func (s *Server) GetSelection() map[string]any {
	s.selMu.RLock()
	defer s.selMu.RUnlock()
	return s.selection
}

// SetSelection stores a browser selection payload for MCP tool retrieval.
func (s *Server) SetSelection(sel map[string]any) {
	s.selMu.Lock()
	defer s.selMu.Unlock()
	s.selection = sel
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleMarbleAPI returns the inner PipelineDef for a composite marble node.
// GET /api/marble/NODE_NAME
func (s *Server) handleMarbleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodeName := strings.TrimPrefix(r.URL.Path, "/api/marble/")
	if nodeName == "" {
		http.Error(w, `{"error":"node name required"}`, http.StatusBadRequest)
		return
	}

	if s.cfg.Marbles == nil {
		http.Error(w, `{"error":"no marble registry configured"}`, http.StatusNotFound)
		return
	}

	factory, ok := s.cfg.Marbles[nodeName]
	if !ok {
		http.Error(w, `{"error":"marble not found"}`, http.StatusNotFound)
		return
	}

	marble := factory(framework.NodeDef{Name: nodeName})
	if !marble.IsComposite() {
		http.Error(w, `{"error":"node is atomic, not expandable"}`, http.StatusBadRequest)
		return
	}

	def := marble.PipelineDef()
	if def == nil {
		http.Error(w, `{"error":"composite marble has no pipeline definition"}`, http.StatusInternalServerError)
		return
	}

	type marbleNode struct {
		Name    string `json:"name"`
		Element string `json:"element,omitempty"`
	}
	type marbleEdge struct {
		ID   string `json:"id"`
		From string `json:"from"`
		To   string `json:"to"`
	}
	type marblePayload struct {
		Pipeline string       `json:"pipeline"`
		Nodes    []marbleNode `json:"nodes"`
		Edges    []marbleEdge `json:"edges"`
		Start    string       `json:"start"`
		Done     string       `json:"done"`
	}

	payload := marblePayload{
		Pipeline: def.Pipeline,
		Start:    def.Start,
		Done:     def.Done,
	}
	for _, n := range def.Nodes {
		payload.Nodes = append(payload.Nodes, marbleNode{Name: n.Name, Element: n.Element})
	}
	for _, e := range def.Edges {
		payload.Edges = append(payload.Edges, marbleEdge{ID: e.ID, From: e.From, To: e.To})
	}
	json.NewEncoder(w).Encode(payload)
}

// ListenAddr returns the HTTP listener address after the server starts.
// Useful for tests that use port 0.
func (s *Server) ListenAddr() string {
	return s.cfg.addr()
}

// StartOnAvailablePort starts on OS-assigned ports and returns them.
// This is primarily for testing.
func (s *Server) StartOnAvailablePort(ctx context.Context) (httpAddr, wsAddr string, err error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/stream", s.handleSSE)
	mux.HandleFunc("POST /events/click", s.handleBrowserEvent("click"))
	mux.HandleFunc("POST /events/hover", s.handleBrowserEvent("hover"))
	mux.HandleFunc("POST /events/selection", s.handleBrowserEvent("selection"))
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/theme", s.handleThemeAPI)
	mux.HandleFunc("GET /api/pipeline", s.handlePipelineAPI)
	mux.HandleFunc("GET /api/kabuki", s.handleKabukiAPI)
	mux.HandleFunc("GET /api/marble/", s.handleMarbleAPI)

	if s.cfg.SPA != nil {
		mux.Handle("GET /", http.FileServer(s.cfg.SPA))
	} else {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Kami debugger running.")
		})
	}

	httpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", fmt.Errorf("HTTP listen: %w", err)
	}

	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/", s.handleWS)
	wsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		httpLn.Close()
		return "", "", fmt.Errorf("WS listen: %w", err)
	}

	s.http = &http.Server{Handler: mux}
	s.ws = &http.Server{Handler: wsMux}

	go s.http.Serve(httpLn)
	go s.ws.Serve(wsLn)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.http.Shutdown(shutCtx)
		s.ws.Shutdown(shutCtx)
	}()

	return httpLn.Addr().String(), wsLn.Addr().String(), nil
}
