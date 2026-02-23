# Origami Pipeline Studio — Product Spec

**Status:** Design-only (no implementation code)
**Contract:** `origami-pipeline-studio`
**Prerequisite:** `origami-fan-out-fan-in` (complete), `origami-network-dispatch` (complete)

## Product Definition

Pipeline Studio is a **separate web product** for real-time pipeline visualization, debugging, and artifact inspection. It is a read-only observer: Studio watches pipeline execution but never controls it.

### Boundary Rules

- Studio lives in its own repository (`github.com/dpopsuev/pipeline-studio`).
- Studio depends on Origami's observer interfaces.
- Origami has **zero dependencies** on Studio. The integration point is the `WalkObserver` interface.
- Studio is optional: pipelines run identically with or without Studio connected.

## StudioObserver Adapter

The `StudioObserver` is an Origami `WalkObserver` implementation that streams events to the Studio backend.

```go
// StudioObserver implements framework.WalkObserver and streams events to a
// Studio backend via HTTP/WebSocket. Included in the Studio repo, not Origami.
type StudioObserver struct {
    endpoint string
    client   *http.Client
    runID    string
    buffer   chan framework.WalkEvent
}

func (o *StudioObserver) OnEvent(e framework.WalkEvent) {
    // Non-blocking send to buffer channel.
    // Buffer is drained by a background goroutine that POSTs to Studio API.
    select {
    case o.buffer <- e:
    default:
        // Drop event if buffer full (Studio is non-critical)
    }
}
```

### Integration Example

```go
studio := studioobserver.New("http://studio.local:3000", runID)
opts := []framework.RunOption{
    framework.WithTransformers(transformers),
    framework.WithObserver(studio),
}
framework.Run(ctx, "pipeline.yaml", input, opts...)
```

## API Contract

REST API. All endpoints are read-only except the event ingestion webhook.

### Resources

#### Runs

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/runs` | Create a run record (called by StudioObserver on walk start) |
| `GET` | `/api/v1/runs` | List runs with pagination and filtering |
| `GET` | `/api/v1/runs/:id` | Get run details |
| `GET` | `/api/v1/runs/:id/events` | Get events for a run (paginated, filterable by type) |
| `GET` | `/api/v1/runs/:id/events/stream` | SSE stream of live events for a run |

#### Events

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/events` | Ingest a batch of WalkEvents (from StudioObserver) |

#### Artifacts

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/runs/:id/artifacts` | List artifacts for a run |
| `GET` | `/api/v1/runs/:id/artifacts/:node` | Get artifact content for a specific node |

### Request/Response Schemas

```json
// POST /api/v1/events
{
  "run_id": "run-abc123",
  "events": [
    {
      "type": "node_enter",
      "node": "recall",
      "walker": "Herald",
      "timestamp": "2026-02-23T20:00:01Z"
    },
    {
      "type": "node_exit",
      "node": "recall",
      "walker": "Herald",
      "elapsed_ms": 1200,
      "artifact": { "type": "rca", "confidence": 0.87 },
      "timestamp": "2026-02-23T20:00:02Z"
    }
  ]
}
```

```json
// GET /api/v1/runs/:id
{
  "id": "run-abc123",
  "pipeline": "rca-investigation",
  "status": "running",
  "start_time": "2026-02-23T20:00:00Z",
  "completion_time": null,
  "current_node": "triage",
  "nodes_visited": ["recall", "triage"],
  "event_count": 42
}
```

## Event Store Schema

SQLite for single-instance deployments; PostgreSQL for production.

### Tables

```sql
CREATE TABLE runs (
    id          TEXT PRIMARY KEY,
    pipeline    TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'running',  -- running, succeeded, failed
    start_time  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time    TIMESTAMP,
    metadata    JSONB       -- pipeline vars, overrides, environment
);

CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES runs(id),
    type        TEXT NOT NULL,       -- node_enter, node_exit, transition, etc.
    node        TEXT,
    walker      TEXT,
    edge        TEXT,
    elapsed_ms  INTEGER,
    error       TEXT,
    metadata    JSONB,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_run_id ON events(run_id);
CREATE INDEX idx_events_type ON events(run_id, type);

CREATE TABLE artifacts (
    id          BIGSERIAL PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES runs(id),
    node        TEXT NOT NULL,
    type        TEXT NOT NULL,
    confidence  REAL,
    raw         JSONB,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(run_id, node)
);
```

## Technology Recommendations

| Component | Recommendation | Rationale |
|-----------|---------------|-----------|
| **Backend** | Go + `net/http` + `chi` router | Same language as Origami; minimal deps; fast |
| **Database** | SQLite (dev) / PostgreSQL (prod) | SQLite for zero-config; PG for concurrent access |
| **Real-time** | Server-Sent Events (SSE) | Simpler than WebSocket; unidirectional is sufficient |
| **Frontend** | React + TypeScript + Vite | Mature ecosystem; rich graph visualization libs |
| **Graph rendering** | `@xyflow/react` (React Flow) | Interactive node-edge graphs; custom node rendering |
| **Timeline** | `react-chrono` or custom | Event timeline with expandable details |
| **Styling** | Tailwind CSS | Utility-first; consistent design without CSS framework overhead |

### Trade-offs Considered

| Option | Pros | Cons | Decision |
|--------|------|------|----------|
| **WebSocket** vs SSE | Bidirectional | More complex; Studio is read-only | SSE |
| **GraphQL** vs REST | Flexible queries | Over-engineering for this domain | REST |
| **Mermaid** vs React Flow | Simpler, text-based | Static, no interaction | React Flow for live view; Mermaid for export |
| **Electron** vs Web | Desktop native | Heavier; web is sufficient | Web |

## UI Wireframes

### Pipeline Graph View (main screen)

```
┌──────────────────────────────────────────────────────────┐
│  Pipeline: rca-investigation          Run: run-abc123    │
│  Status: ● Running                    Elapsed: 00:01:23  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│   ┌─────────┐    E1     ┌─────────┐    E2     ┌──────┐ │
│   │ recall  │ ────────> │ triage  │ ────────> │ done │  │
│   │ ✓ 1.2s  │           │ ● 0.8s  │           │      │  │
│   │ 0.87    │           │  ...    │           │      │  │
│   └─────────┘           └─────────┘           └──────┘  │
│                                                          │
│   Fan-out visualization:                                 │
│                        ┌─────────┐                       │
│                   ┌──> │   B1    │ ──┐                   │
│   ┌─────────┐    │    │ ✓ 0.5s  │   │    ┌─────────┐   │
│   │    A    │ ───┤    └─────────┘   ├──> │    C    │    │
│   │ ✓ 0.2s │    │    ┌─────────┐   │    │ ● ...   │    │
│   └─────────┘    └──> │   B2    │ ──┘    └─────────┘    │
│                       │ ✓ 0.3s  │                        │
│                       └─────────┘                        │
├──────────────────────────────────────────────────────────┤
│  Node: recall  |  Walker: Herald  |  Element: fire       │
│  Confidence: 0.87  |  Artifact type: rca                 │
│  [View Artifact]  [View Prompt]                          │
└──────────────────────────────────────────────────────────┘
```

### Run Timeline View

```
┌──────────────────────────────────────────────────────────┐
│  Timeline: run-abc123                    Filter: [All ▼] │
├──────────────────────────────────────────────────────────┤
│  20:00:00  ○ walk_start                                  │
│  20:00:01  ● node_enter   recall    Herald               │
│  20:00:02  ● node_exit    recall    Herald   1.2s  0.87  │
│  20:00:02  → transition   E1        recall → triage      │
│  20:00:02  ● node_enter   triage    Seeker               │
│  20:00:02  ↔ walker_switch Herald → Seeker               │
│  20:00:03  ● node_exit    triage    Seeker   0.8s  0.92  │
│  20:00:03  → transition   E2        triage → _done       │
│  20:00:03  ✓ walk_complete                               │
├──────────────────────────────────────────────────────────┤
│  ▼ Expand: node_exit recall                              │
│    Walker: Herald                                        │
│    Elapsed: 1.2s                                         │
│    Artifact: { "type": "rca", "confidence": 0.87, ... }  │
└──────────────────────────────────────────────────────────┘
```

### Artifact Inspector

```
┌──────────────────────────────────────────────────────────┐
│  Artifact: recall (run-abc123)              [Raw] [Tree] │
├──────────────────────────────────────────────────────────┤
│  {                                                       │
│    "type": "rca",                                        │
│    "confidence": 0.87,                                   │
│    "raw": {                                              │
│      "root_cause": "timing regression in ptp4l",         │
│      "evidence": [                                       │
│        { "source": "log", "line": 142, ... },            │
│        { "source": "commit", "sha": "abc123", ... }      │
│      ],                                                  │
│      "recommended_actions": [                            │
│        "bisect commits abc123..def456",                   │
│        "check ptp4l config drift"                         │
│      ]                                                   │
│    }                                                     │
│  }                                                       │
└──────────────────────────────────────────────────────────┘
```

## MVP Scope

### In Scope

- Pipeline graph visualization with node status (pending/running/succeeded/failed)
- Live event streaming via SSE
- Run list with search/filter
- Artifact viewer (JSON tree + raw)
- Fan-out/parallel branch visualization
- Event timeline with expandable details

### Non-Goals (future)

- Pipeline editing (Studio is read-only)
- Pipeline execution control (start/stop/retry)
- Multi-user auth (single-user MVP)
- Deployment orchestration (Studio observes, doesn't deploy)
- Historical analytics / dashboards
- Multi-tenant / SaaS mode
