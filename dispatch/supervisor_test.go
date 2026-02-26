package dispatch_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/origami/dispatch"
)

func TestSupervisor_WorkerLifecycle(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus)

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w2"})

	sup.Process()
	h := sup.Health()

	if h.TotalActive != 2 {
		t.Errorf("expected 2 active workers, got %d", h.TotalActive)
	}
	if len(h.Workers) != 2 {
		t.Errorf("expected 2 workers, got %d", len(h.Workers))
	}

	bus.Emit("worker_stopped", "worker", "", "", map[string]string{"worker_id": "w1"})
	sup.Process()
	h = sup.Health()

	if h.TotalActive != 1 {
		t.Errorf("expected 1 active worker, got %d", h.TotalActive)
	}
	if h.TotalStopped != 1 {
		t.Errorf("expected 1 stopped worker, got %d", h.TotalStopped)
	}
}

func TestSupervisor_ErrorThreshold_FlagsReplacement(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus, dispatch.WithErrorThreshold(2))

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	bus.Emit("error", "worker", "C1", "F0", map[string]string{
		"worker_id": "w1",
		"error":     "first error",
	})

	sup.Process()
	h := sup.Health()

	if len(h.ShouldReplace) != 0 {
		t.Errorf("1 error should not trigger replacement, got %v", h.ShouldReplace)
	}

	bus.Emit("error", "worker", "C2", "F0", map[string]string{
		"worker_id": "w1",
		"error":     "second error",
	})

	sup.Process()
	h = sup.Health()

	if h.TotalErrored != 1 {
		t.Errorf("expected 1 errored worker, got %d", h.TotalErrored)
	}
	if len(h.ShouldReplace) != 1 || h.ShouldReplace[0] != "w1" {
		t.Errorf("expected [w1] in should_replace, got %v", h.ShouldReplace)
	}
}

func TestSupervisor_SilenceThreshold_FlagsReplacement(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus, dispatch.WithSilenceThreshold(50*time.Millisecond))

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	sup.Process()

	time.Sleep(100 * time.Millisecond)

	h := sup.Health()
	if len(h.ShouldReplace) != 1 || h.ShouldReplace[0] != "w1" {
		t.Errorf("expected [w1] flagged as silent, got %v", h.ShouldReplace)
	}
}

func TestSupervisor_StepCounting(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus)

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	bus.Emit("done", "worker", "C1", "F0", map[string]string{"worker_id": "w1"})
	bus.Emit("done", "worker", "C2", "F0", map[string]string{"worker_id": "w1"})
	bus.Emit("done", "worker", "C3", "F0", map[string]string{"worker_id": "w1"})

	sup.Process()
	h := sup.Health()

	for _, w := range h.Workers {
		if w.WorkerID == "w1" {
			if w.StepsComplete != 3 {
				t.Errorf("expected 3 steps complete, got %d", w.StepsComplete)
			}
			return
		}
	}
	t.Error("worker w1 not found in health summary")
}

func TestSupervisor_ShouldStop(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus)

	if sup.ShouldStop() {
		t.Error("should_stop should be false initially")
	}

	sup.EmitShouldStop()
	sup.Process()

	if !sup.ShouldStop() {
		t.Error("should_stop should be true after EmitShouldStop")
	}
}

func TestSupervisor_BudgetTracking(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus, dispatch.WithBudgetTotal(1000))

	bus.Emit("budget_update", "system", "", "", map[string]string{"used": "500"})
	sup.Process()

	h := sup.Health()
	if h.BudgetUsedPct < 49.9 || h.BudgetUsedPct > 50.1 {
		t.Errorf("expected ~50%% budget used, got %.1f%%", h.BudgetUsedPct)
	}
}

func TestSupervisor_IncrementalProcessing(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus)

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	sup.Process()

	h := sup.Health()
	if h.TotalActive != 1 {
		t.Fatalf("expected 1 active, got %d", h.TotalActive)
	}

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w2"})
	sup.Process()

	h = sup.Health()
	if h.TotalActive != 2 {
		t.Errorf("expected 2 active after incremental process, got %d", h.TotalActive)
	}
}

// TestSupervisor_ConcurrentProcess_Race exposes two consequences of
// reading lastProcessed outside the mutex in Process():
//
//  1. Double-counting: multiple goroutines get the same signal batch,
//     each incrementing StepsComplete for the same events.
//  2. Signal blindness: lastProcessed overshoots past the actual signal
//     count, so bus.Since(N) returns nil and the tracker never sees
//     signals emitted after the race.
//
// This test fails deterministically under -race (data race on
// lastProcessed) and with very high probability without -race
// (observable logical corruption).
func TestSupervisor_ConcurrentProcess_Race(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus)

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	const doneSignals = 5
	for i := 0; i < doneSignals; i++ {
		bus.Emit("done", "worker", fmt.Sprintf("C%d", i), "F0",
			map[string]string{"worker_id": "w1"})
	}

	// Fire 50 goroutines at Process() simultaneously.
	// A barrier ensures they all read lastProcessed before any can write it.
	const goroutines = 50
	var barrier sync.WaitGroup
	barrier.Add(goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			barrier.Done()
			barrier.Wait()
			sup.Process()
		}()
	}
	wg.Wait()

	h := sup.Health()
	for _, w := range h.Workers {
		if w.WorkerID == "w1" && w.StepsComplete != doneSignals {
			t.Errorf("double-counting: expected %d steps, got %d",
				doneSignals, w.StepsComplete)
		}
	}

	// Emit a new signal and verify Process() still sees it.
	// If lastProcessed overshot, bus.Since(overshot) returns nil
	// and this signal is silently lost.
	bus.Emit("done", "worker", "C_late", "F0",
		map[string]string{"worker_id": "w1"})
	sup.Process()

	h = sup.Health()
	for _, w := range h.Workers {
		if w.WorkerID == "w1" {
			if w.StepsComplete != doneSignals+1 {
				t.Errorf("signal blindness: expected %d steps after late signal, got %d",
					doneSignals+1, w.StepsComplete)
			}
			return
		}
	}
	t.Error("w1 not found in health summary")
}

func TestSupervisor_MultipleWorkersIndependent(t *testing.T) {
	bus := dispatch.NewSignalBus()
	sup := dispatch.NewSupervisorTracker(bus, dispatch.WithErrorThreshold(2))

	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w1"})
	bus.Emit("worker_started", "worker", "", "", map[string]string{"worker_id": "w2"})
	bus.Emit("error", "worker", "C1", "F0", map[string]string{"worker_id": "w1", "error": "e1"})
	bus.Emit("error", "worker", "C2", "F0", map[string]string{"worker_id": "w1", "error": "e2"})
	bus.Emit("done", "worker", "C3", "F0", map[string]string{"worker_id": "w2"})

	sup.Process()
	h := sup.Health()

	if h.TotalErrored != 1 {
		t.Errorf("expected 1 errored, got %d", h.TotalErrored)
	}
	if h.TotalActive != 1 {
		t.Errorf("expected 1 active, got %d", h.TotalActive)
	}

	var w1found, w2found bool
	for _, w := range h.Workers {
		switch w.WorkerID {
		case "w1":
			w1found = true
			if w.Status != "errored" {
				t.Errorf("w1 expected errored, got %s", w.Status)
			}
			if w.ErrorCount != 2 {
				t.Errorf("w1 expected 2 errors, got %d", w.ErrorCount)
			}
		case "w2":
			w2found = true
			if w.Status != "active" {
				t.Errorf("w2 expected active, got %s", w.Status)
			}
			if w.StepsComplete != 1 {
				t.Errorf("w2 expected 1 step, got %d", w.StepsComplete)
			}
		}
	}
	if !w1found || !w2found {
		t.Error("missing worker in health summary")
	}
}
