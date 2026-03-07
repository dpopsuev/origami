package framework
// Category: Processing & Support

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// narrationSink receives a single human-readable narration line.
type narrationSink func(line string)

// narrationOption configures a narrationObserver.
type narrationOption func(*narrationObserver)

// withVocabulary sets the vocabulary for translating node/edge names.
func withVocabulary(v Vocabulary) narrationOption {
	return func(n *narrationObserver) { n.vocab = v }
}

// withSink sets the output destination for narration lines.
func withSink(s narrationSink) narrationOption {
	return func(n *narrationObserver) { n.sink = s }
}

// withMilestoneInterval sets how often milestone summaries are emitted.
// A value of 0 disables milestones. Default is 5.
func withMilestoneInterval(every int) narrationOption {
	return func(n *narrationObserver) { n.milestoneEvery = every }
}

// withETA enables or disables ETA estimation in narration output.
func withETA(enabled bool) narrationOption {
	return func(n *narrationObserver) { n.showETA = enabled }
}

// progress captures a snapshot of walk progress.
type progress struct {
	NodesVisited int
	Elapsed      time.Duration
	CurrentNode  string
	LastWalker   string
}

// narrationObserver is a WalkObserver that produces human-readable narration
// lines from walk events. It translates node names via a Vocabulary, tracks
// progress, computes ETA, and emits milestone summaries.
//
// Zero-config: newNarrationObserver() with no options logs to slog.Info.
type narrationObserver struct {
	mu             sync.Mutex
	vocab          Vocabulary
	sink           narrationSink
	milestoneEvery int
	showETA        bool

	walkStart    time.Time
	nodesVisited int
	currentNode  string
	lastWalker   string
	errors       int
}

// newNarrationObserver creates a narration observer with sensible defaults.
// Pass narrationOption values to customize vocabulary, sink, etc.
func newNarrationObserver(opts ...narrationOption) *narrationObserver {
	n := &narrationObserver{
		vocab:          VocabularyFunc(func(code string) string { return code }),
		sink:           func(line string) { slog.Info(line) },
		milestoneEvery: 5,
		showETA:        true,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// progress returns a snapshot of current walk progress.
func (n *narrationObserver) progress() progress {
	n.mu.Lock()
	defer n.mu.Unlock()
	elapsed := time.Duration(0)
	if !n.walkStart.IsZero() {
		elapsed = time.Since(n.walkStart)
	}
	return progress{
		NodesVisited: n.nodesVisited,
		Elapsed:      elapsed,
		CurrentNode:  n.currentNode,
		LastWalker:   n.lastWalker,
	}
}

// OnEvent implements WalkObserver.
func (n *narrationObserver) OnEvent(e WalkEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch e.Type {
	case EventNodeEnter:
		if n.walkStart.IsZero() {
			n.walkStart = time.Now()
		}
		n.currentNode = e.Node
		if e.Walker != "" {
			n.lastWalker = e.Walker
		}
		name := n.vocab.Name(e.Node)
		if e.Walker != "" {
			n.emit(fmt.Sprintf("[%s] Entering %s", e.Walker, name))
		} else {
			n.emit(fmt.Sprintf("Entering %s", name))
		}

	case EventNodeExit:
		n.nodesVisited++
		name := n.vocab.Name(e.Node)
		if e.Error != nil {
			n.errors++
			n.emit(fmt.Sprintf("Failed at %s: %v", name, e.Error))
		} else if e.Elapsed > 0 {
			n.emit(fmt.Sprintf("Completed %s (%s)", name, fmtNarrateDuration(e.Elapsed)))
		} else {
			n.emit(fmt.Sprintf("Completed %s", name))
		}
		if n.milestoneEvery > 0 && n.nodesVisited%n.milestoneEvery == 0 {
			n.emitMilestone()
		}

	case EventWalkerSwitch:
		n.lastWalker = e.Walker
		name := n.vocab.Name(e.Node)
		n.emit(fmt.Sprintf("Handing off to %s at %s", e.Walker, name))

	case EventTransition:
		// silent by default; transitions are high-frequency noise

	case EventEdgeEvaluate:
		// silent by default

	case EventWalkComplete:
		elapsed := time.Since(n.walkStart)
		n.emit(fmt.Sprintf("Walk complete — %d nodes visited in %s",
			n.nodesVisited, fmtNarrateDuration(elapsed)))

	case EventWalkError:
		n.errors++
		node := e.Node
		if node != "" {
			node = n.vocab.Name(node)
		}
		if node != "" {
			n.emit(fmt.Sprintf("Walk failed at %s: %v", node, e.Error))
		} else {
			n.emit(fmt.Sprintf("Walk failed: %v", e.Error))
		}
	}
}

func (n *narrationObserver) emit(line string) {
	n.sink(line)
}

func (n *narrationObserver) emitMilestone() {
	elapsed := time.Since(n.walkStart)
	line := fmt.Sprintf("--- progress: %d nodes visited | Elapsed: %s",
		n.nodesVisited, fmtNarrateDuration(elapsed))
	if n.showETA && n.nodesVisited > 0 {
		avgPerNode := elapsed / time.Duration(n.nodesVisited)
		line += fmt.Sprintf(" | Avg: %s/node", fmtNarrateDuration(avgPerNode))
	}
	if n.errors > 0 {
		line += fmt.Sprintf(" | Errors: %d", n.errors)
	}
	line += " ---"
	n.emit(line)
}

func fmtNarrateDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	s := d.Seconds()
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	m := int(s) / 60
	sec := int(s) % 60
	return fmt.Sprintf("%dm%ds", m, sec)
}
