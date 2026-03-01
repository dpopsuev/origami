# Origami K8s Pipeline Operator — Design Document

**Status:** Design-only (no implementation code)
**Contract:** `origami-k8s-pipeline-operator`
**Prerequisite:** `origami-fan-out-fan-in` (complete), `origami-network-dispatch` (complete)

## Overview

The same `PipelineDef` YAML that runs locally via `origami run` becomes a Kubernetes Custom Resource Definition (CRD). Nodes execute as Jobs or Pods; edges trigger as events evaluated by a controller. This is the BYOI capstone: infrastructure-agnostic pipelines from laptop to cluster.

## CRD Schema

Two resources: `Pipeline` (definition) and `PipelineRun` (execution instance).

### Pipeline

```yaml
apiVersion: origami.io/v1alpha1
kind: Pipeline
metadata:
  name: rca-investigation
spec:
  pipeline: rca-investigation
  vars:
    recall_hit: "0.85"
  nodes:
    - name: recall
      transformer: llm
      image: registry.example.com/rca-agent:latest
      resources:
        limits:
          memory: 512Mi
          cpu: "1"
      prompt: prompts/recall.md
    - name: triage
      transformer: llm
      image: registry.example.com/rca-agent:latest
      input: "${recall.output}"
      prompt: prompts/triage.md
  edges:
    - id: E1
      from: recall
      to: triage
      when: "output.match == true"
    - id: E2
      from: triage
      to: _done
      when: "true"
  start: recall
  done: _done
```

The `spec` section maps 1:1 to `PipelineDef` fields with two Kubernetes-specific additions per node: `image` (container image) and `resources` (K8s resource limits).

### PipelineRun

```yaml
apiVersion: origami.io/v1alpha1
kind: PipelineRun
metadata:
  name: rca-run-001
  ownerReferences:
    - apiVersion: origami.io/v1alpha1
      kind: Pipeline
      name: rca-investigation
spec:
  pipelineRef: rca-investigation
  overrides:
    recall_hit: "0.9"
status:
  phase: Running  # Pending | Running | Succeeded | Failed
  currentNode: triage
  startTime: "2026-02-23T20:00:00Z"
  completionTime: null
  nodes:
    - name: recall
      phase: Succeeded
      jobName: rca-run-001-recall
      artifact: recall-artifact-configmap
      startTime: "2026-02-23T20:00:01Z"
      completionTime: "2026-02-23T20:00:45Z"
    - name: triage
      phase: Running
      jobName: rca-run-001-triage
```

## Controller Architecture

### Reconciliation Loop

```
Watch PipelineRun
  -> If phase == Pending:
       Validate Pipeline spec, create initial status, set phase = Running
  -> If phase == Running:
       1. Find currentNode
       2. Check if Job for currentNode exists
          - No: Create Job (see Job Template below)
          - Yes, Running: Requeue after 10s
          - Yes, Succeeded: Read artifact, evaluate edges, advance currentNode
          - Yes, Failed: Set PipelineRun phase = Failed
       3. If currentNode == _done: Set phase = Succeeded
  -> If phase == Succeeded | Failed:
       No-op (terminal)
```

### Job Template

Each node becomes a Kubernetes Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: <pipelinerun>-<node>
  labels:
    origami.io/pipeline: <pipeline>
    origami.io/run: <pipelinerun>
    origami.io/node: <node>
spec:
  backoffLimit: 2
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        readOnlyRootFilesystem: true
      containers:
        - name: node
          image: <node.image>
          resources: <node.resources>
          env:
            - name: ORIGAMI_NODE
              value: <node.name>
            - name: ORIGAMI_DISPATCH_URL
              value: http://origami-controller.<namespace>.svc:8080
            - name: ORIGAMI_DISPATCH_ID
              valueFrom:
                fieldRef:
                  fieldPath: metadata.annotations['origami.io/dispatch-id']
          volumeMounts:
            - name: artifacts
              mountPath: /artifacts
      volumes:
        - name: artifacts
          emptyDir: {}
      restartPolicy: Never
```

## Artifact Transport

### Evaluation

| Mechanism | Pros | Cons | Recommendation |
|-----------|------|------|----------------|
| **ConfigMap** | Simple, native K8s, no external deps | 1MB limit, not suitable for large artifacts | Default for small artifacts (<256KB) |
| **PVC shared volume** | No size limit, fast local access | Requires ReadWriteMany, complex lifecycle | For large artifacts with shared storage |
| **S3/Object storage** | Unlimited size, durable, multi-cluster | External dependency, latency | Production-grade, multi-cluster |
| **NetworkDispatcher (HTTP)** | Already implemented, proven protocol | Requires controller to run HTTP server | Recommended first implementation |

**Recommendation:** Use `NetworkDispatcher` (HTTP) as the primary mechanism. The controller runs a `NetworkServer`; each Job pod runs a `NetworkClient`. This reuses proven code and the existing `ExternalDispatcher` protocol. ConfigMap fallback for artifact metadata/status.

## Edge Evaluation

**Central (controller-side)** — recommended.

The controller reads the artifact after Job completion and evaluates the `when:` expression using the same `expr-lang/expr` engine as local execution. This keeps the evaluation logic in one place and avoids distributing the expression engine to every node container.

Distributed (sidecar) evaluation was considered but rejected: it adds a sidecar to every pod, requires shipping the expr engine, and creates consistency risks when different pods run different versions.

## State Management

**CRD status subresource** — recommended.

Walker state maps to `PipelineRun.status`:
- `currentNode` -> `status.currentNode`
- `Outputs` -> per-node artifact references in `status.nodes[].artifact`
- `History` -> per-node phase/timing in `status.nodes[]`
- `LoopCounts` -> `status.loopCounts` map

The status subresource is the standard K8s pattern for tracking runtime state. No ConfigMaps needed for state.

## Security

| Concern | Mitigation |
|---------|------------|
| **RBAC** | Controller ServiceAccount: create/delete Jobs, read/update Pipelines and PipelineRuns. Namespace-scoped by default. |
| **Pod Security** | Default securityContext: `runAsNonRoot: true`, `readOnlyRootFilesystem: true`, no added capabilities. |
| **Network** | NetworkPolicy restricts pod-to-controller traffic to the dispatch port only. |
| **Secrets** | Pipeline vars that contain secrets use `secretKeyRef`, never plaintext in CRD spec. |
| **TLS** | Controller HTTP server uses TLS. Pod-to-controller traffic encrypted in transit. |

## Migration Path

| Stage | Execution | What changes |
|-------|-----------|-------------|
| **Local** | `origami run pipeline.yaml` | Nothing — current behavior |
| **Remote agents** | `NetworkServer` + `NetworkClient` | Add network transport, same pipeline YAML |
| **Kubernetes** | `kubectl apply -f pipeline.yaml` | Add `image:` and `resources:` to nodes, wrap in CRD |

The pipeline YAML is the constant. Only infrastructure-specific fields are added when moving to K8s. The `spec` section is a superset of `PipelineDef`.

## Open Questions (for implementation phase)

1. **Multi-cluster:** Should PipelineRun support cross-cluster node scheduling?
2. **Retry policy:** Per-node retry vs pipeline-level retry? Current design: Job `backoffLimit`.
3. **Observability:** OpenTelemetry integration? WalkObserver -> OTEL exporter?
4. **Garbage collection:** Auto-cleanup of completed Jobs? TTL-based or owner-reference cascade?
5. **Admission webhook:** Validate Pipeline CRDs at admission time using `Validate()`?
