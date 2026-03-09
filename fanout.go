package framework

// Category: Execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// parallelMatch pairs a matched edge with its transition during fan-out detection.
type parallelMatch struct {
	edge       Edge
	transition *Transition
}

// isParallelEdge returns true if the edge implements ParallelEdge and is marked parallel.
func isParallelEdge(e Edge) bool {
	if pe, ok := e.(ParallelEdge); ok {
		return pe.IsParallel()
	}
	return false
}

// walkFanOut executes parallel branch nodes concurrently and returns the merge node.
// Each parallel edge target is processed in its own goroutine. If any branch fails,
// remaining branches are cancelled via the errgroup context.
//
// After all branches complete, outgoing edges from each branch target are evaluated
// to find the common merge node. All branches must agree on the same successor.
//
// Returns (mergeNodeName, error). The caller handles the done-node case.
func (g *DefaultGraph) walkFanOut(
	ctx context.Context,
	walker Walker,
	obs WalkObserver,
	sourceNode Node,
	sourceArtifact Artifact,
	matches []parallelMatch,
) (string, error) {
	state := walker.State()
	walkerName := walker.Identity().PersonaName

	branchNames := make([]string, len(matches))
	for i, m := range matches {
		branchNames[i] = m.transition.NextNode
	}
	emitEvent(obs, WalkEvent{
		Type:     EventFanOutStart,
		Node:     sourceNode.Name(),
		Walker:   walkerName,
		Metadata: map[string]any{"branches": branchNames},
	})

	results := make([]branchResult, len(matches))
	var outputMu sync.Mutex

	eg, egCtx := errgroup.WithContext(ctx)

	for i, m := range matches {
		targetNode, ok := g.nodeIndex[m.transition.NextNode]
		if !ok {
			return "", fmt.Errorf("%w: fan-out target %q from edge %s",
				ErrNodeNotFound, m.transition.NextNode, m.edge.ID())
		}

		eg.Go(func() error {
			emitEvent(obs, WalkEvent{Type: EventNodeEnter, Node: targetNode.Name(), Walker: walkerName})
			start := time.Now()

			nc := NodeContext{
				WalkerState:   state,
				PriorArtifact: sourceArtifact,
				Meta:          make(map[string]any),
			}

			branchCtx, branchCancel := g.nodeCtx(egCtx, targetNode.Name())
			defer branchCancel()

			art, err := walker.Handle(branchCtx, targetNode, nc)
			elapsed := time.Since(start)

			if err != nil {
				emitEvent(obs, WalkEvent{
					Type: EventNodeExit, Node: targetNode.Name(),
					Walker: walkerName, Elapsed: elapsed, Error: err,
				})
				return fmt.Errorf("node %s: %w", targetNode.Name(), err)
			}

			emitEvent(obs, WalkEvent{
				Type: EventNodeExit, Node: targetNode.Name(),
				Walker: walkerName, Artifact: art, Elapsed: elapsed,
			})

			outputMu.Lock()
			if state.Outputs == nil {
				state.Outputs = make(map[string]Artifact)
			}
			state.Outputs[targetNode.Name()] = art
			outputMu.Unlock()

			results[i] = branchResult{nodeName: targetNode.Name(), artifact: art}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		state.Status = "error"
		emitEvent(obs, WalkEvent{Type: EventWalkError, Node: sourceNode.Name(), Error: err})
		return "", err
	}

	for _, m := range matches {
		state.RecordStep(sourceNode.Name(), m.edge.ID(), m.edge.ID(),
			time.Now().UTC().Format(time.RFC3339))
	}

	mergeNodeName, err := g.findMergeTarget(results, state)
	if err != nil {
		state.Status = "error"
		emitEvent(obs, WalkEvent{Type: EventWalkError, Node: sourceNode.Name(), Error: err})
		return "", err
	}

	emitEvent(obs, WalkEvent{
		Type:     EventFanOutEnd,
		Node:     sourceNode.Name(),
		Walker:   walkerName,
		Metadata: map[string]any{"merge": mergeNodeName},
	})

	return mergeNodeName, nil
}

// findMergeTarget evaluates outgoing edges from each parallel branch and returns
// the common successor node. All branches must converge on the same target.
func (g *DefaultGraph) findMergeTarget(results []branchResult, state *WalkerState) (string, error) {
	var mergeNodeName string

	for _, r := range results {
		edges := g.EdgesFrom(r.nodeName)
		var found string
		for _, e := range edges {
			t := e.Evaluate(r.artifact, state)
			if t != nil {
				found = t.NextNode
				break
			}
		}
		if found == "" {
			return "", fmt.Errorf("%w: branch %q has no matching outgoing edge",
				ErrFanOutMerge, r.nodeName)
		}
		if mergeNodeName == "" {
			mergeNodeName = found
		} else if mergeNodeName != found {
			return "", fmt.Errorf("%w: branches disagree on merge target: %q vs %q",
				ErrFanOutMerge, mergeNodeName, found)
		}
	}

	if mergeNodeName == "" {
		return "", fmt.Errorf("%w: no merge node found", ErrFanOutMerge)
	}

	return mergeNodeName, nil
}

// branchResult holds the output of a single parallel branch for merge-target detection.
type branchResult struct {
	nodeName string
	artifact Artifact
}

// ListArtifact wraps multiple artifacts from parallel branches into a single
// composite artifact. Produced by the "append" merge strategy.
type ListArtifact struct {
	Items []Artifact
}

func (a *ListArtifact) Type() string       { return "list" }
func (a *ListArtifact) Confidence() float64 { return 0 }
func (a *ListArtifact) Raw() any            { return a.Items }

// applyMergeStrategy combines branch results into a single merged artifact
// based on the merge strategy declared on the edge leading to the merge node.
func applyMergeStrategy(strategy string, results []branchResult) Artifact {
	if len(results) == 0 {
		return nil
	}
	switch strategy {
	case MergeAppend:
		items := make([]Artifact, 0, len(results))
		for _, r := range results {
			if r.artifact != nil {
				items = append(items, r.artifact)
			}
		}
		return &ListArtifact{Items: items}
	case MergeLatest:
		return results[len(results)-1].artifact
	default:
		return results[0].artifact
	}
}
