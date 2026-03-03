package sumi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/view"
)

const (
	minBackoff = 100 * time.Millisecond
	maxBackoff = 5 * time.Second
)

// sseClientLoop connects to a Kami SSE endpoint and feeds events into
// the store. On disconnect (e.g. session swap), it reconnects with
// exponential backoff and re-bootstraps from /api/snapshot so the client
// picks up the new circuit definition and clears stale walkers.
func sseClientLoop(ctx context.Context, addr string, store *view.CircuitStore, log *slog.Logger) {
	backoff := minBackoff
	first := true
	for {
		if !first {
			rebootstrapStore(addr, store, log)
		}
		first = false

		err := streamSSE(ctx, addr, store)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Debug("SSE stream ended", "error", err, "reconnect_in", backoff)
		} else {
			log.Debug("SSE stream closed by server", "reconnect_in", backoff)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff = backoff * 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// rebootstrapStore fetches /api/snapshot from Kami and resets the client
// store with the new circuit definition. This clears stale walkers and
// picks up any new session's node set.
func rebootstrapStore(addr string, store *view.CircuitStore, log *slog.Logger) {
	url := fmt.Sprintf("http://%s/api/snapshot", addr)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Debug("re-bootstrap snapshot unavailable", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("re-bootstrap snapshot non-200", "status", resp.StatusCode)
		return
	}

	var snap view.CircuitSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		log.Debug("re-bootstrap snapshot decode failed", "error", err)
		return
	}

	def := &framework.CircuitDef{Circuit: snap.CircuitName}
	for name := range snap.Nodes {
		def.Nodes = append(def.Nodes, framework.NodeDef{Name: name})
	}

	store.Reset(def)

	for name, ns := range snap.Nodes {
		if ns.State == view.NodeActive || ns.State == view.NodeCompleted || ns.State == view.NodeError {
			var evtType framework.WalkEventType
			switch ns.State {
			case view.NodeActive:
				evtType = framework.EventNodeEnter
			case view.NodeCompleted:
				evtType = framework.EventNodeExit
			case view.NodeError:
				evtType = framework.EventWalkError
			}
			store.OnEvent(framework.WalkEvent{Type: evtType, Node: name})
		}
	}

	for walkerID, wp := range snap.Walkers {
		store.OnEvent(framework.WalkEvent{
			Type:   framework.EventNodeEnter,
			Node:   wp.Node,
			Walker: walkerID,
		})
	}

	log.Debug("re-bootstrapped store from snapshot",
		"circuit", snap.CircuitName,
		"nodes", len(snap.Nodes),
		"walkers", len(snap.Walkers))
}

func streamSSE(ctx context.Context, addr string, store *view.CircuitStore) error {
	url := fmt.Sprintf("http://%s/events/stream", addr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := line[len("data: "):]

		var evt kami.Event
		if err := json.Unmarshal([]byte(payload), &evt); err != nil {
			continue
		}

		we := eventToWalkEvent(evt)
		store.OnEvent(we)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read: %w", err)
	}
	return nil
}
