package framework

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// BatchCase represents a single case in a batch walk.
type BatchCase struct {
	ID       string
	Context  map[string]any
	Adapters []*Adapter
}

// BatchWalkResult captures the outcome of walking one case.
type BatchWalkResult struct {
	CaseID        string
	Path          []string
	StepArtifacts map[string]Artifact
	State         *WalkerState
	Error         error
}

// BatchWalkConfig configures a batch walk over a circuit.
type BatchWalkConfig struct {
	Def            *CircuitDef
	Shared         GraphRegistries
	Cases          []BatchCase
	Parallel       int
	OnCaseComplete func(index int, result BatchWalkResult)
}

// BatchWalk walks a circuit once per case, optionally in parallel.
// Each case gets its own runner (shared registries + per-case adapters),
// walker, and observer. Results are returned in case order.
func BatchWalk(ctx context.Context, cfg BatchWalkConfig) []BatchWalkResult {
	results := make([]BatchWalkResult, len(cfg.Cases))

	walkOne := func(ctx context.Context, i int, bc BatchCase) {
		reg := cfg.Shared
		if len(bc.Adapters) > 0 {
			var err error
			reg, err = MergeAdapters(reg, bc.Adapters...)
			if err != nil {
				results[i] = BatchWalkResult{CaseID: bc.ID, Error: err}
				return
			}
		}

		runner, err := NewRunnerWith(cfg.Def, reg)
		if err != nil {
			results[i] = BatchWalkResult{CaseID: bc.ID, Error: err}
			return
		}

		walker := NewProcessWalker(bc.ID)
		walker.State().MergeContext(bc.Context)

		var mu sync.Mutex
		var path []string
		stepArtifacts := map[string]Artifact{}

		obs := WalkObserverFunc(func(e WalkEvent) {
			mu.Lock()
			defer mu.Unlock()
			if e.Type == EventNodeEnter {
				path = append(path, e.Node)
			}
			if e.Type == EventNodeExit && e.Artifact != nil {
				stepArtifacts[e.Node] = e.Artifact
			}
		})
		if dg, ok := runner.Graph.(*DefaultGraph); ok {
			dg.SetObserver(obs)
		}

		walkErr := runner.Walk(ctx, walker, cfg.Def.Start)

		results[i] = BatchWalkResult{
			CaseID:        bc.ID,
			Path:          path,
			StepArtifacts: stepArtifacts,
			State:         walker.State(),
			Error:         walkErr,
		}
	}

	if cfg.Parallel > 1 {
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(cfg.Parallel)
		for i, bc := range cfg.Cases {
			i, bc := i, bc
			g.Go(func() error {
				walkOne(gCtx, i, bc)
				if cfg.OnCaseComplete != nil {
					cfg.OnCaseComplete(i, results[i])
				}
				return nil
			})
		}
		_ = g.Wait()
	} else {
		for i, bc := range cfg.Cases {
			walkOne(ctx, i, bc)
			if cfg.OnCaseComplete != nil {
				cfg.OnCaseComplete(i, results[i])
			}
		}
	}

	return results
}
