package sumi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/kami"
	"github.com/dpopsuev/origami/view"

	tea "github.com/charmbracelet/bubbletea"
)

// RunConfig holds the configuration for the `origami sumi` command.
type RunConfig struct {
	CircuitPath string
	KamiAddr    string
	WatchAddr   string
	ReplayFile  string
	NoColor     bool
	Compact     bool
}

// Run starts the Sumi TUI.
// In normal mode it loads a circuit YAML and renders it statically.
// In --watch mode it connects to a running Kami SSE stream.
// In --replay mode it plays back a recorded JSONL file.
func Run(ctx context.Context, cfg RunConfig) error {
	if cfg.ReplayFile != "" {
		return runReplay(ctx, cfg)
	}
	if cfg.WatchAddr != "" {
		return runWatch(ctx, cfg)
	}
	return runCircuit(ctx, cfg)
}

func runCircuit(_ context.Context, cfg RunConfig) error {
	if cfg.CircuitPath == "" {
		return fmt.Errorf("circuit path required")
	}

	data, err := os.ReadFile(cfg.CircuitPath)
	if err != nil {
		return fmt.Errorf("read circuit: %w", err)
	}
	def, err := framework.LoadCircuit(data)
	if err != nil {
		return fmt.Errorf("load circuit: %w", err)
	}

	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, err := engine.Layout(def)
	if err != nil {
		return fmt.Errorf("layout: %w", err)
	}

	opts := RenderOpts{NoColor: cfg.NoColor, Compact: cfg.Compact}

	if cfg.NoColor {
		// Non-interactive mode: render once and exit
		snap := store.Snapshot()
		fmt.Print(RenderGraph(def, layout, snap, opts))
		fmt.Print(renderStatusLine(snap, opts))
		return nil
	}

	var debug *DebugClient
	if cfg.KamiAddr != "" {
		debug = NewDebugClient(cfg.KamiAddr)
	}

	recorder := NewViewRecorder(defaultRecorderCapacity)

	m := New(Config{
		Def:      def,
		Store:    store,
		Layout:   layout,
		Opts:     opts,
		Debug:    debug,
		Recorder: recorder,
	})

	if debug != nil && debug.HealthCheck() {
		m.kamiStatus = KamiConnected
		m.debugAvail = true
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("sumi: %w", err)
	}
	return nil
}

func runWatch(ctx context.Context, cfg RunConfig) error {
	addr := cfg.WatchAddr
	log := slog.Default().With("component", "sumi-sse")

	var debug *DebugClient
	if cfg.KamiAddr != "" {
		debug = NewDebugClient(cfg.KamiAddr)
	} else {
		debug = NewDebugClient(addr)
	}

	def, store := bootstrapFromSnapshot(addr, log)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	recorder := NewViewRecorder(defaultRecorderCapacity)

	m := New(Config{
		Def:      def,
		Store:    store,
		Layout:   layout,
		Opts:     RenderOpts{NoColor: cfg.NoColor, Compact: cfg.Compact},
		Debug:    debug,
		Recorder: recorder,
	})

	if debug.HealthCheck() {
		m.kamiStatus = KamiConnected
		m.debugAvail = true
	}

	sseCtx, sseCancel := context.WithCancel(ctx)
	defer sseCancel()
	go sseClientLoop(sseCtx, addr, store, log)

	pushCtx, pushCancel := context.WithCancel(ctx)
	defer pushCancel()
	go framePushLoop(pushCtx, recorder, addr, log)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("sumi watch: %w", err)
	}
	return nil
}

const framePushInterval = 500 * time.Millisecond

// framePushLoop polls the recorder and POSTs new frames to Kami.
// Runs at most 2 pushes/second. Errors are silently ignored.
func framePushLoop(ctx context.Context, rec *ViewRecorder, kamiAddr string, log *slog.Logger) {
	url := fmt.Sprintf("http://%s/api/sumi/frame", kamiAddr)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastTS time.Time

	ticker := time.NewTicker(framePushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f := rec.Latest()
			if f == nil || f.Timestamp.Equal(lastTS) {
				continue
			}
			lastTS = f.Timestamp

			body, err := json.Marshal(f)
			if err != nil {
				continue
			}
			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				log.Debug("frame push failed", "error", err)
				continue
			}
			resp.Body.Close()
		}
	}
}

// bootstrapFromSnapshot fetches /api/snapshot from Kami to build the
// CircuitDef and CircuitStore with the correct node set. Falls back to
// an empty def if Kami is unreachable (SSE will populate walkers later).
func bootstrapFromSnapshot(addr string, log *slog.Logger) (*framework.CircuitDef, *view.CircuitStore) {
	url := fmt.Sprintf("http://%s/api/snapshot", addr)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Debug("snapshot unavailable, starting with empty circuit", "error", err)
		def := &framework.CircuitDef{Circuit: "watch"}
		return def, view.NewCircuitStore(def)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debug("snapshot returned non-200, starting with empty circuit", "status", resp.StatusCode)
		def := &framework.CircuitDef{Circuit: "watch"}
		return def, view.NewCircuitStore(def)
	}

	var snap view.CircuitSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		log.Debug("snapshot decode failed, starting with empty circuit", "error", err)
		def := &framework.CircuitDef{Circuit: "watch"}
		return def, view.NewCircuitStore(def)
	}

	def := &framework.CircuitDef{Circuit: snap.CircuitName}
	for name := range snap.Nodes {
		def.Nodes = append(def.Nodes, framework.NodeDef{Name: name})
	}

	store := view.NewCircuitStore(def)

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

	log.Info("bootstrapped from snapshot", "circuit", snap.CircuitName, "nodes", len(snap.Nodes), "walkers", len(snap.Walkers))
	return def, store
}

func runReplay(_ context.Context, cfg RunConfig) error {
	f, err := os.Open(cfg.ReplayFile)
	if err != nil {
		return fmt.Errorf("open replay: %w", err)
	}
	defer f.Close()

	def := &framework.CircuitDef{Circuit: "replay"}
	store := view.NewCircuitStore(def)
	defer store.Close()

	bridge := kami.NewEventBridge(nil)
	defer bridge.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	opts := RenderOpts{NoColor: cfg.NoColor, Compact: cfg.Compact}

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   opts,
	})

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		scanner := bufio.NewScanner(f)
		var prevTS time.Time
		for scanner.Scan() {
			var evt kami.Event
			if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
				continue
			}
			if !prevTS.IsZero() && !evt.Timestamp.IsZero() {
				delay := evt.Timestamp.Sub(prevTS)
				if delay > 0 && delay < 10*time.Second {
					time.Sleep(delay)
				}
			}
			prevTS = evt.Timestamp

			we := eventToWalkEvent(evt)
			store.OnEvent(we)
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("sumi replay: %w", err)
	}
	return nil
}

func eventToWalkEvent(evt kami.Event) framework.WalkEvent {
	we := framework.WalkEvent{
		Node:   evt.Node,
		Walker: evt.Agent,
		Edge:   evt.Edge,
	}
	switch evt.Type {
	case kami.EventNodeEnter:
		we.Type = framework.EventNodeEnter
	case kami.EventNodeExit:
		we.Type = framework.EventNodeExit
	case kami.EventTransition:
		we.Type = framework.EventTransition
	case kami.EventWalkComplete:
		we.Type = framework.EventWalkComplete
	case kami.EventWalkError:
		we.Type = framework.EventWalkError
		we.Error = fmt.Errorf("%s", evt.Error)
	case kami.EventFanOutStart:
		we.Type = framework.EventFanOutStart
	case kami.EventFanOutEnd:
		we.Type = framework.EventFanOutEnd
	default:
		we.Type = framework.WalkEventType(evt.Type)
	}
	return we
}

func renderStatusLine(snap view.CircuitSnapshot, opts RenderOpts) string {
	parts := []string{fmt.Sprintf("Circuit: %s", snap.CircuitName)}
	parts = append(parts, fmt.Sprintf("Nodes: %d", len(snap.Nodes)))
	if snap.Completed {
		parts = append(parts, "[DONE]")
	}
	if snap.Error != "" {
		parts = append(parts, fmt.Sprintf("[ERROR: %s]", snap.Error))
	}
	_ = opts
	return fmt.Sprintln(strings.Join(parts, "  │  "))
}
