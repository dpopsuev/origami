package sumi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   opts,
		Debug:  debug,
	})

	if debug != nil && debug.HealthCheck() {
		m.kamiStatus = KamiConnected
		m.debugAvail = true
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("sumi: %w", err)
	}
	return nil
}

func runWatch(_ context.Context, cfg RunConfig) error {
	addr := cfg.WatchAddr
	def := &framework.CircuitDef{Circuit: "watch"}
	store := view.NewCircuitStore(def)
	defer store.Close()

	engine := &view.GridLayout{}
	layout, _ := engine.Layout(def)

	var debug *DebugClient
	if cfg.KamiAddr != "" {
		debug = NewDebugClient(cfg.KamiAddr)
	} else {
		debug = NewDebugClient(addr)
	}

	m := New(Config{
		Def:    def,
		Store:  store,
		Layout: layout,
		Opts:   RenderOpts{NoColor: cfg.NoColor, Compact: cfg.Compact},
		Debug:  debug,
	})

	if debug.HealthCheck() {
		m.kamiStatus = KamiConnected
		m.debugAvail = true
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("sumi watch: %w", err)
	}
	return nil
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

	p := tea.NewProgram(m, tea.WithAltScreen())

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
