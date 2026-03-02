package sumi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DebugClient is an HTTP client for Kami's Debug API.
// It sends debug commands (pause, resume, step, breakpoints) to a running
// Kami server. When the Kami server is unavailable, methods return errors
// silently — Sumi degrades gracefully.
type DebugClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDebugClient creates a client pointing at a Kami HTTP server.
func NewDebugClient(kamiAddr string) *DebugClient {
	if !strings.HasPrefix(kamiAddr, "http") {
		kamiAddr = "http://" + kamiAddr
	}
	return &DebugClient{
		baseURL: strings.TrimRight(kamiAddr, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// HealthCheck returns true if the Kami server is reachable.
func (d *DebugClient) HealthCheck() bool {
	resp, err := d.httpClient.Get(d.baseURL + "/api/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// SetBreakpoint sets a breakpoint on the named node.
func (d *DebugClient) SetBreakpoint(node string) error {
	return d.postJSON("/api/debug/breakpoint", map[string]string{"node": node, "action": "set"})
}

// ClearBreakpoint clears a breakpoint from the named node.
func (d *DebugClient) ClearBreakpoint(node string) error {
	return d.postJSON("/api/debug/breakpoint", map[string]string{"node": node, "action": "clear"})
}

// Pause pauses the circuit walk.
func (d *DebugClient) Pause() error {
	return d.postJSON("/api/debug/pause", nil)
}

// Resume resumes a paused circuit walk.
func (d *DebugClient) Resume() error {
	return d.postJSON("/api/debug/resume", nil)
}

// AdvanceNode steps to the next node.
func (d *DebugClient) AdvanceNode() error {
	return d.postJSON("/api/debug/advance", nil)
}

func (d *DebugClient) postJSON(path string, payload any) error {
	var body *strings.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		body = strings.NewReader(string(data))
	} else {
		body = strings.NewReader("{}")
	}

	resp, err := d.httpClient.Post(d.baseURL+path, "application/json", body)
	if err != nil {
		return fmt.Errorf("debug request %s: %w", path, err)
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("debug %s: status %d", path, resp.StatusCode)
	}
	return nil
}
