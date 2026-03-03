package kami

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestStart_PortOccupied_ReturnsError(t *testing.T) {
	// Occupy the HTTP port before Kami starts.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer blocker.Close()
	port := blocker.Addr().(*net.TCPAddr).Port

	bridge := NewEventBridge(nil)
	defer bridge.Close()

	srv := NewServer(Config{
		Port:   port,
		Bridge: bridge,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error from Start when port is occupied, got nil")
		}
		t.Logf("Start returned error as expected: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return error within 5s — silent failure")
	}
}

func TestStart_PortOccupied_WSServerCleansUp(t *testing.T) {
	// Occupy the HTTP port so HTTP fails, but WS port (port+1) is free.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer blocker.Close()
	port := blocker.Addr().(*net.TCPAddr).Port

	bridge := NewEventBridge(nil)
	defer bridge.Close()

	srv := NewServer(Config{
		Port:   port,
		Bridge: bridge,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return")
	}

	// WS port (port+1) should NOT still be listening after Start returns.
	wsAddr := fmt.Sprintf("127.0.0.1:%d", port+1)
	conn, err := net.DialTimeout("tcp", wsAddr, 500*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Errorf("WS server on %s still listening after HTTP failure — resource leak", wsAddr)
	}
}

func TestStart_SSEUnavailable_WhenHTTPFailsToBind(t *testing.T) {
	// This replicates the exact production scenario:
	// 1. A stale process occupies port 3001
	// 2. asterisk serve starts Kami on port 3001
	// 3. HTTP fails silently, WS succeeds on 3002
	// 4. Sumi connects to 3001 and gets WebSocket protocol violation instead of SSE

	// Occupy the HTTP port with a dummy server.
	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := blocker.Addr().(*net.TCPAddr).Port

	dummySrv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})}
	go dummySrv.Serve(blocker)
	defer dummySrv.Close()

	bridge := NewEventBridge(nil)
	defer bridge.Close()

	srv := NewServer(Config{
		Port:   port,
		Bridge: bridge,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("Start should have returned an error for occupied port")
		}
		t.Logf("correctly detected port conflict: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Start hung instead of failing fast")
	}

	// The SSE endpoint should NOT be available on the occupied port
	// (the dummy server returns 418, not Kami's SSE stream).
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/events/stream", port))
	if err != nil {
		t.Logf("connection to port %d failed (expected if blocker closed): %v", port, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("SSE endpoint returned 200 on the occupied port — Kami should not be serving here")
	}
}

func TestStart_BothPortsFree_ServesSSE(t *testing.T) {
	bridge := NewEventBridge(nil)
	defer bridge.Close()

	srv := NewServer(Config{Bridge: bridge})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpAddr, _, err := srv.StartOnAvailablePort(ctx)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/api/health", httpAddr))
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health returned %d, want 200", resp.StatusCode)
	}
}
