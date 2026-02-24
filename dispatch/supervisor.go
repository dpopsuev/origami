package dispatch

import (
	"sync"
	"time"
)

// WorkerState tracks the health state of a single worker.
type WorkerState struct {
	WorkerID      string    `json:"worker_id"`
	Status        string    `json:"status"` // "active", "idle", "errored", "stopped"
	ErrorCount    int       `json:"error_count"`
	StepsComplete int       `json:"steps_complete"`
	LastSeen      time.Time `json:"last_seen"`
	LastError     string    `json:"last_error,omitempty"`
}

// HealthSummary is a snapshot of all tracked workers and overall pipeline health.
type HealthSummary struct {
	Workers        []WorkerState `json:"workers"`
	TotalActive    int           `json:"total_active"`
	TotalErrored   int           `json:"total_errored"`
	TotalStopped   int           `json:"total_stopped"`
	ShouldReplace  []string      `json:"should_replace,omitempty"`
	ShouldStop     bool          `json:"should_stop"`
	QueueDepth     int           `json:"queue_depth,omitempty"`
	BudgetUsedPct  float64       `json:"budget_used_pct,omitempty"`
}

// SupervisorTracker watches a SignalBus and maintains per-worker health state.
// The supervisor agent queries this for health summaries to make replacement
// and shutdown decisions.
type SupervisorTracker struct {
	mu               sync.Mutex
	workers          map[string]*WorkerState
	lastProcessed    int
	bus              *SignalBus
	silenceThreshold time.Duration
	errorThreshold   int
	shouldStop       bool
	budgetTotal      float64
	budgetUsed       float64
}

// SupervisorOption configures a SupervisorTracker.
type SupervisorOption func(*SupervisorTracker)

// WithSilenceThreshold sets how long a worker can be silent before being
// flagged for replacement. Default: 2 minutes.
func WithSilenceThreshold(d time.Duration) SupervisorOption {
	return func(s *SupervisorTracker) { s.silenceThreshold = d }
}

// WithErrorThreshold sets how many errors a worker can accumulate before being
// flagged for replacement. Default: 3.
func WithErrorThreshold(n int) SupervisorOption {
	return func(s *SupervisorTracker) { s.errorThreshold = n }
}

// WithBudgetTotal sets the total budget for budget tracking (arbitrary units).
func WithBudgetTotal(total float64) SupervisorOption {
	return func(s *SupervisorTracker) { s.budgetTotal = total }
}

// NewSupervisorTracker creates a health tracker that watches the given SignalBus.
func NewSupervisorTracker(bus *SignalBus, opts ...SupervisorOption) *SupervisorTracker {
	s := &SupervisorTracker{
		workers:          make(map[string]*WorkerState),
		bus:              bus,
		silenceThreshold: 2 * time.Minute,
		errorThreshold:   3,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Process reads new signals from the bus and updates worker state.
func (s *SupervisorTracker) Process() {
	signals := s.bus.Since(s.lastProcessed)
	if len(signals) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sig := range signals {
		s.lastProcessed++
		wid := sig.Meta["worker_id"]
		if wid == "" && sig.Event != "should_stop" && sig.Event != "budget_update" {
			continue
		}

		switch sig.Event {
		case "worker_started":
			s.workers[wid] = &WorkerState{
				WorkerID: wid,
				Status:   "active",
				LastSeen: s.parseTime(sig.Timestamp),
			}

		case "worker_stopped":
			if w, ok := s.workers[wid]; ok {
				w.Status = "stopped"
				w.LastSeen = s.parseTime(sig.Timestamp)
			}

		case "start", "done":
			if w, ok := s.workers[wid]; ok {
				w.LastSeen = s.parseTime(sig.Timestamp)
				w.Status = "active"
				if sig.Event == "done" {
					w.StepsComplete++
				}
			}

		case "error":
			if w, ok := s.workers[wid]; ok {
				w.ErrorCount++
				w.LastError = sig.Meta["error"]
				w.LastSeen = s.parseTime(sig.Timestamp)
				if w.ErrorCount >= s.errorThreshold {
					w.Status = "errored"
				}
			}

		case "should_stop":
			s.shouldStop = true

		case "budget_update":
			if v, ok := sig.Meta["used"]; ok {
				n, _ := parseFloat(v)
				s.budgetUsed = n
			}
		}
	}
}

// Health returns a snapshot of the current worker health state.
// Callers should call Process() first to ensure state is up-to-date.
func (s *SupervisorTracker) Health() HealthSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	summary := HealthSummary{
		ShouldStop: s.shouldStop,
	}

	if s.budgetTotal > 0 {
		summary.BudgetUsedPct = (s.budgetUsed / s.budgetTotal) * 100
	}

	for _, w := range s.workers {
		summary.Workers = append(summary.Workers, *w)

		switch w.Status {
		case "active":
			summary.TotalActive++
			if s.silenceThreshold > 0 && now.Sub(w.LastSeen) > s.silenceThreshold {
				summary.ShouldReplace = append(summary.ShouldReplace, w.WorkerID)
			}
		case "errored":
			summary.TotalErrored++
			summary.ShouldReplace = append(summary.ShouldReplace, w.WorkerID)
		case "stopped":
			summary.TotalStopped++
		}
	}

	return summary
}

// EmitShouldStop emits a should_stop signal on the bus, instructing workers
// to finish their current step and exit.
func (s *SupervisorTracker) EmitShouldStop() {
	s.bus.Emit("should_stop", "supervisor", "", "", nil)
}

// ShouldStop returns true if a should_stop signal has been processed.
func (s *SupervisorTracker) ShouldStop() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shouldStop
}

func (s *SupervisorTracker) parseTime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Now()
	}
	return t
}

func parseFloat(s string) (float64, bool) {
	var result float64
	var decimal float64
	var inDecimal bool
	for _, c := range s {
		if c >= '0' && c <= '9' {
			if inDecimal {
				decimal /= 10
				result += float64(c-'0') * decimal
			} else {
				result = result*10 + float64(c-'0')
			}
		} else if c == '.' {
			inDecimal = true
			decimal = 1
		}
	}
	return result, true
}
