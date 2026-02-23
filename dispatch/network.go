package dispatch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// NetworkServer wraps an ExternalDispatcher and exposes it over HTTP.
// Agents connect and poll for work via GET /next, then submit results via POST /submit.
type NetworkServer struct {
	dispatcher ExternalDispatcher
	server     *http.Server
	log        *slog.Logger
	addr       string
	mu         sync.Mutex
	started    bool
}

// NetworkServerOption configures a NetworkServer.
type NetworkServerOption func(*NetworkServer)

// WithTLS configures TLS for the network server.
func WithTLS(cfg *tls.Config) NetworkServerOption {
	return func(s *NetworkServer) {
		s.server.TLSConfig = cfg
	}
}

// WithServerLogger sets the logger for the network server.
func WithServerLogger(l *slog.Logger) NetworkServerOption {
	return func(s *NetworkServer) { s.log = l }
}

// NewNetworkServer creates an HTTP server that exposes an ExternalDispatcher.
func NewNetworkServer(dispatcher ExternalDispatcher, addr string, opts ...NetworkServerOption) *NetworkServer {
	s := &NetworkServer{
		dispatcher: dispatcher,
		log:        discardLogger(),
		addr:       addr,
		server:     &http.Server{Addr: addr},
	}
	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /next", s.handleGetNext)
	mux.HandleFunc("POST /submit", s.handleSubmit)
	s.server.Handler = mux

	return s
}

// Serve starts the HTTP server and blocks until the context is cancelled or
// the server encounters a fatal error.
func (s *NetworkServer) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("network server listen: %w", err)
	}

	s.mu.Lock()
	s.addr = ln.Addr().String()
	s.started = true
	s.mu.Unlock()

	s.log.Info("network server started", slog.String("addr", s.addr))

	go func() {
		<-ctx.Done()
		s.server.Close()
	}()

	if s.server.TLSConfig != nil {
		tlsLn := tls.NewListener(ln, s.server.TLSConfig)
		err = s.server.Serve(tlsLn)
	} else {
		err = s.server.Serve(ln)
	}

	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Addr returns the address the server is listening on.
// Only valid after Serve has been called.
func (s *NetworkServer) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

type nextResponse struct {
	DispatchID   int64  `json:"dispatch_id"`
	CaseID       string `json:"case_id"`
	Step         string `json:"step"`
	PromptPath   string `json:"prompt_path"`
	ArtifactPath string `json:"artifact_path"`
}

type submitRequest struct {
	DispatchID int64  `json:"dispatch_id"`
	Data       []byte `json:"data"`
}

func (s *NetworkServer) handleGetNext(w http.ResponseWriter, r *http.Request) {
	dc, err := s.dispatcher.GetNextStep(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	resp := nextResponse{
		DispatchID:   dc.DispatchID,
		CaseID:       dc.CaseID,
		Step:         dc.Step,
		PromptPath:   dc.PromptPath,
		ArtifactPath: dc.ArtifactPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *NetworkServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.dispatcher.SubmitArtifact(r.Context(), req.DispatchID, req.Data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NetworkClient implements ExternalDispatcher by calling a remote NetworkServer
// over HTTP. This is the agent-side counterpart to NetworkServer.
type NetworkClient struct {
	baseURL string
	client  *http.Client
	log     *slog.Logger
}

// NetworkClientOption configures a NetworkClient.
type NetworkClientOption func(*NetworkClient)

// WithNetworkHTTPClient sets a custom HTTP client (for auth middleware, TLS, timeouts).
func WithNetworkHTTPClient(c *http.Client) NetworkClientOption {
	return func(nc *NetworkClient) { nc.client = c }
}

// WithClientLogger sets the logger for the network client.
func WithClientLogger(l *slog.Logger) NetworkClientOption {
	return func(nc *NetworkClient) { nc.log = l }
}

// NewNetworkClient creates an ExternalDispatcher that connects to a remote
// NetworkServer. The baseURL should be like "http://localhost:8080".
func NewNetworkClient(baseURL string, opts ...NetworkClientOption) *NetworkClient {
	nc := &NetworkClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Minute},
		log:     discardLogger(),
	}
	for _, opt := range opts {
		opt(nc)
	}
	return nc
}

// GetNextStep polls the server for the next dispatch context.
func (c *NetworkClient) GetNextStep(ctx context.Context) (DispatchContext, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/next", nil)
	if err != nil {
		return DispatchContext{}, fmt.Errorf("network client: create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return DispatchContext{}, fmt.Errorf("network client: GET /next: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return DispatchContext{}, fmt.Errorf("network client: GET /next: status %d: %s",
			resp.StatusCode, string(body))
	}

	var nr nextResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		return DispatchContext{}, fmt.Errorf("network client: decode response: %w", err)
	}

	return DispatchContext{
		DispatchID:   nr.DispatchID,
		CaseID:       nr.CaseID,
		Step:         nr.Step,
		PromptPath:   nr.PromptPath,
		ArtifactPath: nr.ArtifactPath,
	}, nil
}

// SubmitArtifact sends artifact data to the server for the given dispatch ID.
func (c *NetworkClient) SubmitArtifact(ctx context.Context, dispatchID int64, data []byte) error {
	body, err := json.Marshal(submitRequest{DispatchID: dispatchID, Data: data})
	if err != nil {
		return fmt.Errorf("network client: marshal submit: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/submit",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("network client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("network client: POST /submit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("network client: POST /submit: status %d: %s",
			resp.StatusCode, string(respBody))
	}

	return nil
}

var _ ExternalDispatcher = (*NetworkClient)(nil)
