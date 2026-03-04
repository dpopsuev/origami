package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/schematics/rca/scenarios"
	"github.com/spf13/cobra"
)

var (
	consumeProject      string
	consumeLookbackDays int
	consumeCandidateDir string
	consumeDatasetDir   string
	consumeDryRun       bool
	consumeRPBase       string
	consumeRPKeyPath    string
)

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Data ingestion commands",
	Long:  "Discover new CI failures and create candidate cases for dataset growth.",
}

var consumeRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Walk the ingestion circuit to discover new failures",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		scenario, err := scenarios.LoadScenario("ptp-real-ingest")
		if err != nil {
			return fmt.Errorf("load scenario: %w", err)
		}
		symptoms := scenario.Symptoms

		dedupIdx, err := LoadDedupIndex(consumeDatasetDir, consumeCandidateDir)
		if err != nil {
			return fmt.Errorf("load dedup index: %w", err)
		}

		var discoverer RunDiscoverer
		if consumeDryRun {
			fmt.Fprintln(cmd.OutOrStdout(), "[dry-run] Using stub discoverer (no RP API calls)")
			discoverer = &stubDiscoverer{}
		} else {
			rpBase := consumeRPBase
			if rpBase == "" {
				rpBase = os.Getenv("ASTERISK_RP_URL")
			}
			if rpBase == "" {
				return fmt.Errorf("RP base URL required: set --rp-base-url or $ASTERISK_RP_URL")
			}
			if cfg.discovererFactory == nil {
				return fmt.Errorf("no run discoverer configured (discoverer factory not injected)")
			}
			d, err := cfg.discovererFactory(rpBase, consumeRPKeyPath, consumeProject)
			if err != nil {
				return fmt.Errorf("create run discoverer: %w", err)
			}
			discoverer = d
		}

		nodeReg := IngestNodeRegistry(
			discoverer, symptoms, consumeProject, dedupIdx, consumeCandidateDir,
		)

		circuitData, err := os.ReadFile("internal/circuits/asterisk-ingest.yaml")
		if err != nil {
			return fmt.Errorf("read circuit: %w", err)
		}

		def, err := framework.LoadCircuit(circuitData)
		if err != nil {
			return fmt.Errorf("parse circuit: %w", err)
		}

		edgeIDs := make([]string, len(def.Edges))
		for i, ed := range def.Edges {
			edgeIDs[i] = ed.ID
		}
		edgeFactory := make(framework.EdgeFactory, len(edgeIDs))
		for _, id := range edgeIDs {
			edgeFactory[id] = func(ed framework.EdgeDef) framework.Edge {
				return &consumeForwardEdge{def: ed}
			}
		}

		reg := framework.GraphRegistries{
			Nodes: nodeReg,
			Edges: edgeFactory,
		}

		graph, err := def.BuildGraph(reg)
		if err != nil {
			return fmt.Errorf("build graph: %w", err)
		}

		walker := framework.NewProcessWalker("consume")
		walker.State().Context["config"] = &IngestConfig{
			RPProject:    consumeProject,
			LookbackDays: consumeLookbackDays,
			DatasetDir:   consumeDatasetDir,
			CandidateDir: consumeCandidateDir,
		}

		if err := graph.Walk(ctx, walker, def.Start); err != nil {
			return fmt.Errorf("walk circuit: %w", err)
		}

		if summary, ok := walker.State().Context["summary"]; ok {
			if s, ok := summary.(IngestSummary); ok {
				fmt.Fprintf(cmd.OutOrStdout(),
					"Fetched %d launches, found %d failures, matched %d symptoms, "+
						"created %d candidates (%d deduplicated)\n",
					s.LaunchesFetched, s.FailuresParsed, s.SymptomsMatched,
					s.CandidatesCreated, s.Deduplicated)
			}
		}

		return nil
	},
}

type consumeForwardEdge struct {
	def framework.EdgeDef
}

func (e *consumeForwardEdge) ID() string         { return e.def.ID }
func (e *consumeForwardEdge) From() string       { return e.def.From }
func (e *consumeForwardEdge) To() string         { return e.def.To }
func (e *consumeForwardEdge) IsShortcut() bool   { return e.def.Shortcut }
func (e *consumeForwardEdge) IsLoop() bool       { return e.def.Loop }
func (e *consumeForwardEdge) Evaluate(_ framework.Artifact, _ *framework.WalkerState) *framework.Transition {
	return &framework.Transition{NextNode: e.def.To}
}

type stubDiscoverer struct{}

func (f *stubDiscoverer) DiscoverRuns(_ string, _ time.Time) ([]RunInfo, error) {
	return nil, nil
}
func (f *stubDiscoverer) FetchFailures(_ int) ([]FailureInfo, error) {
	return nil, nil
}

func init() {
	consumeRunCmd.Flags().StringVar(&consumeProject, "project", "", "RP project name")
	consumeRunCmd.Flags().IntVar(&consumeLookbackDays, "lookback", 7, "Days to look back for launches")
	consumeRunCmd.Flags().StringVar(&consumeCandidateDir, "candidate-dir", "candidates", "Directory for candidate case files")
	consumeRunCmd.Flags().StringVar(&consumeDatasetDir, "dataset-dir", "datasets", "Directory for verified dataset files")
	consumeRunCmd.Flags().BoolVar(&consumeDryRun, "dry-run", false, "Use stub fetcher (no RP API calls)")
	consumeRunCmd.Flags().StringVar(&consumeRPBase, "rp-base-url", "", "RP base URL (default: $ASTERISK_RP_URL)")
	consumeRunCmd.Flags().StringVar(&consumeRPKeyPath, "rp-api-key", ".rp-api-key", "Path to RP API key file")

	consumeCmd.AddCommand(consumeRunCmd)
}
