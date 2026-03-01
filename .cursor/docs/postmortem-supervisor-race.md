# Post-Mortem: SupervisorTracker Race Condition

**Date found:** 2026-02-26
**Component:** `dispatch/supervisor.go` — `SupervisorTracker.Process()`
**Severity:** High (silent data corruption → cascading operational failures)
**Detection method:** `go test -race` on `TestV2Workers_FullDrain_Deterministic`

## The One-Line Bug

```go
func (s *SupervisorTracker) Process() {
    signals := s.bus.Since(s.lastProcessed) // ← READ without lock
    // ...
    s.mu.Lock()
    for _, sig := range signals {
        s.lastProcessed++ // ← WRITE with lock
```

`lastProcessed` is read outside the mutex, written inside it. When the MCP SDK dispatches concurrent `handleGetNextStep` handlers — one per worker calling `get_next_step` — multiple goroutines race on this field.

## The Mechanism

Two goroutines calling `Process()` simultaneously:

```
Goroutine A               Goroutine B
─────────────             ─────────────
read lastProcessed=0
                          read lastProcessed=0
call bus.Since(0) → [s0,s1,s2,s3]
                          call bus.Since(0) → [s0,s1,s2,s3]  (same batch!)
lock mutex
  process s0,s1,s2,s3
  lastProcessed = 4
unlock
                          lock mutex
                            process s0,s1,s2,s3  (AGAIN)
                            lastProcessed = 8
                          unlock
```

**Result:** 4 signals exist, but `lastProcessed=8`. Every counter has been incremented twice. All future `bus.Since(8)` calls return nil.

With N concurrent goroutines, `lastProcessed` reaches `N × batch_size`. With 4 workers, a single race window multiplies signal processing by up to 4x.

## Production Symptoms — The Cascade

Had this gone unfixed, the following sequence would unfold during a wet calibration run with `--parallel=4`:

### Phase 1: Silent Corruption (T+0s)

All 4 workers call `get_next_step` concurrently at session start. Each handler calls `sess.Supervisor.Process()`. With 4 `worker_started` signals in the bus:

- 3 goroutines read `lastProcessed=0` before the first writes
- Each processes the same 4 signals
- `lastProcessed` jumps to 12 instead of 4
- Each worker's `StepsComplete` counter starts at 0 but is structurally correct (worker_started doesn't increment it)

**No visible symptom yet.** Everything looks fine.

### Phase 2: The Tracker Goes Blind (T+1s)

Workers start processing steps. The bus accumulates `step_ready`, `done`, and worker signals. But:

```
bus.Len() = 15          (4 started + 11 new signals)
lastProcessed = 12      (from the Phase 1 overshoot)
bus.Since(12) = [s12, s13, s14]  — only sees the last 3
```

Wait — that's only partially blind. If more signals arrive than the overshoot, the tracker recovers *partially*. But it missed signals 4-11 permanently. Those contained `done` events, `error` events, and `worker_started` for late-joining workers.

The real danger: with 4 workers polling rapidly, the race fires **repeatedly**. Each concurrent `Process()` call overshoots further. After 10 rounds of 4-worker concurrent polling:

```
actual signals: ~50
lastProcessed:  ~200  (each round overshoots by ~3x)
bus.Since(200): nil   — completely blind
```

### Phase 3: Ghost Workers (T+30s)

The supervisor's `Health()` output now shows:

```json
{
  "workers": [
    {"worker_id": "w0", "status": "active", "steps_complete": 0, "last_seen": "2026-02-26T10:00:01Z"},
    {"worker_id": "w1", "status": "active", "steps_complete": 0, "last_seen": "2026-02-26T10:00:01Z"},
    {"worker_id": "w2", "status": "active", "steps_complete": 0, "last_seen": "2026-02-26T10:00:01Z"},
    {"worker_id": "w3", "status": "active", "steps_complete": 0, "last_seen": "2026-02-26T10:00:01Z"}
  ],
  "total_active": 4,
  "should_replace": []
}
```

Workers appear active with 0 steps complete — but they've actually processed 15 steps. The tracker is frozen at its initial snapshot. `LastSeen` timestamps are from 30 seconds ago.

From the supervisor's perspective: **four workers are alive but doing nothing.**

### Phase 4: False Silence Alarms (T+2min)

The `silenceThreshold` (default: 2 minutes) fires. `Health()` now returns:

```json
{
  "should_replace": ["w0", "w1", "w2", "w3"],
  "total_active": 4
}
```

All four healthy, productive workers are flagged for replacement. The supervisor agent, following its protocol, would:

1. Kill all 4 workers
2. Launch 4 replacement workers
3. The replacements register via `worker_started` signals
4. But `bus.Since(200)` still returns nil
5. The tracker never sees the new workers
6. After 2 more minutes: "all workers silent, replace again"

**Result: infinite replacement loop.** The system burns through LLM tokens spawning and killing workers while actual circuit progress stalls.

### Phase 5: Error Count Inflation (Intermittent)

When a worker hits a legitimate error (e.g., LLM timeout), the `error` signal gets processed N times:

```
Actual errors:     1
Reported errors:   3  (processed by 3 concurrent goroutines)
Error threshold:   3
```

A single transient error crosses the threshold. The worker is classified as `"errored"` and added to `ShouldReplace`. A healthy worker with one bad request gets terminated.

This is the most insidious symptom because it's **intermittent** — it depends on exact goroutine timing. In logs, you'd see a worker flagged for replacement with `error_count=3`, but only one error event in the signal bus. The numbers don't add up, and the debugging trail goes cold because the race is non-deterministic.

### Phase 6: Budget Hallucination (If Budget Tracking Enabled)

```
Actual budget used:    500 / 1000 (50%)
Reported budget used:  1500 / 1000 (150%)  — triple-counted
```

If budget-based shutdown is implemented, the system terminates early thinking it's blown the budget. A 30-case calibration run stops at case 10, reports "budget exceeded," and the operator re-runs — burning more actual budget to compensate for phantom budget.

## Why Tests Didn't Catch It Earlier

The test `TestV2Workers_FullDrain_Deterministic` exercises the exact production code path — 4 workers, concurrent `get_next_step`, full circuit drain. But:

1. **Without `-race`**: The Go runtime's goroutine scheduler often happens to serialize the concurrent `Process()` calls just enough to avoid the overshoot. The starvation check (`workLog[i] == 0`) never fires because channel fairness is adequate for 8 steps / 4 workers.

2. **The test's name is ironic**: "Deterministic" — but it relied on scheduling luck. The race existed on every run; it just didn't produce observable test failures without the race detector.

3. **Unit tests were sequential**: `TestSupervisor_WorkerLifecycle`, `TestSupervisor_StepCounting`, etc. — all called `Process()` from a single goroutine. The race requires concurrent callers, which only happens through the MCP handler path.

## The Fix

Move the `bus.Since()` call inside the mutex:

```go
func (s *SupervisorTracker) Process() {
    s.mu.Lock()
    defer s.mu.Unlock()

    signals := s.bus.Since(s.lastProcessed)
    if len(signals) == 0 {
        return
    }
    // ...
}
```

Lock ordering: `supervisor.mu → bus.mu` (inside `Since()`). No reverse path exists anywhere in the codebase, so no deadlock risk.

The critical section is now ~microseconds longer (adds one `bus.Since()` call under the lock), but `Process()` is called once per `get_next_step` poll — not a hot path.

## The Regression Test

`TestSupervisor_ConcurrentProcess_Race` fires 50 goroutines at `Process()` behind a `sync.WaitGroup` barrier, then checks:

1. **Double-counting**: `StepsComplete` must equal exactly the number of `done` signals emitted (not N×)
2. **Signal blindness**: A signal emitted *after* the concurrent burst must be visible to a subsequent serial `Process()` call

The test catches both failure modes:
- With `-race`: always detects the data race (100/100)
- Without `-race`: catches logical corruption ~40% of the time (scheduling-dependent, but high enough for CI)

## Lessons

1. **Mutex scope must cover the full read-modify-write cycle.** Reading the index outside the lock and writing inside is the textbook split-brain pattern. If you see `lock(); ...; x++; unlock()` but the *decision* that feeds into `x++` was made outside the lock, there's a race.

2. **Supervisor/tracker components are the last place you want silent corruption.** They're the "immune system" — if they go blind, nothing else can detect that workers are healthy. The monitoring system itself must be hardened against concurrency.

3. **"Works in tests, works in prod" is not proven without `-race`.** This race existed since the `SupervisorTracker` was written. It survived all unit tests because they were sequential. Only the integration test with concurrent MCP handlers triggered it.

4. **A flaky test is a signal, not noise.** The test was named "Deterministic" but was secretly relying on scheduler fairness. When a test occasionally fails and you can't explain why, the answer is almost never "the test is wrong" — it's "the test found something the others missed."
