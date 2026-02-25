package probes

import (
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestScoreDebug_PerfectDiagnosis(t *testing.T) {
	response := `1. Root cause: Goroutine leak from the async notification worker (v2.14.0) exhausting the connection pool.
2. Evidence:
   - goroutine count: 12847 (baseline ~200) — 60x increase
   - connection pool exhausted: 0/50 — all connections held by leaked goroutines
   - Deployed v2.14.0 with async notification worker at 13:30
   - Feature flag "async-notifications" enabled at 13:45, errors start at 14:01
3. Red herrings rejected:
   - Memory at 52-56% is not critical and is a symptom of goroutine accumulation, not the root cause
   - GC pause is elevated but is a consequence, not a cause
4. Recommended fix: Add context cancellation to the async notification worker goroutines.`

	scores := ScoreDebug(response)

	if scores[ouroboros.DimConvergenceThreshold] < 0.8 {
		t.Errorf("ConvergenceThreshold = %f, want >= 0.8 (perfect diagnosis)", scores[ouroboros.DimConvergenceThreshold])
	}
	if scores[ouroboros.DimShortcutAffinity] > 0.4 {
		t.Errorf("ShortcutAffinity = %f, want <= 0.4 (rejected red herring)", scores[ouroboros.DimShortcutAffinity])
	}
}

func TestScoreDebug_ShallowDiagnosis(t *testing.T) {
	response := `The root cause is high memory usage at 52%. Need to optimize memory allocation.`

	scores := ScoreDebug(response)

	if scores[ouroboros.DimConvergenceThreshold] > 0.3 {
		t.Errorf("ConvergenceThreshold = %f, want <= 0.3 (wrong root cause)", scores[ouroboros.DimConvergenceThreshold])
	}
	if scores[ouroboros.DimSpeed] < 0.7 {
		t.Errorf("Speed = %f, want >= 0.7 (very brief response)", scores[ouroboros.DimSpeed])
	}
}

func TestScoreDebug_Determinism(t *testing.T) {
	response := `Root cause: goroutine leak from async notification worker causing connection pool exhaustion.`

	s1 := ScoreDebug(response)
	s2 := ScoreDebug(response)

	for _, dim := range []ouroboros.Dimension{ouroboros.DimSpeed, ouroboros.DimShortcutAffinity, ouroboros.DimConvergenceThreshold} {
		if s1[dim] != s2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, s1[dim], s2[dim])
		}
	}
}
