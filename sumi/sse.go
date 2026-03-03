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

	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/view"
)

const (
	minBackoff = 100 * time.Millisecond
	maxBackoff = 5 * time.Second
)

// sseClientLoop connects to a Kami SSE endpoint and feeds events into
// the store. On disconnect (e.g. session swap), it reconnects with
// exponential backoff. Exits when ctx is cancelled.
func sseClientLoop(ctx context.Context, addr string, store *view.CircuitStore, log *slog.Logger) {
	backoff := minBackoff
	for {
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
