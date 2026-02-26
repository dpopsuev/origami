package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// AnalyzeFunc is the consumer's domain function — the reason the tool exists.
type AnalyzeFunc func(ctx context.Context, args []string) error

// DatasetStore manages datasets (list, status, import, export, review, promote).
type DatasetStore interface {
	List() ([]DatasetSummary, error)
	Status(name string) (*DatasetStatus, error)
	Import(path string) error
	Export(path string) error
	ListCandidates() ([]Candidate, error)
	Promote(caseID string) error
}

// DatasetSummary is the short view of a dataset.
type DatasetSummary struct {
	Name       string `json:"name"`
	CaseCount  int    `json:"case_count"`
	Status     string `json:"status"`
}

// DatasetStatus is the detailed view of a dataset.
type DatasetStatus struct {
	Name       string `json:"name"`
	CaseCount  int    `json:"case_count"`
	Reviewed   int    `json:"reviewed"`
	Promoted   int    `json:"promoted"`
}

// Candidate is a case that can be promoted into a dataset.
type Candidate struct {
	CaseID string `json:"case_id"`
	Source string `json:"source"`
}

// CalibrateRunner runs scorecard evaluation against a dataset.
type CalibrateRunner interface {
	Run(ctx context.Context, scenario string) error
	Report(ctx context.Context, scenario string) error
	Compare(ctx context.Context, a, b string) error
}

// ServeConfig configures the MCP server command.
type ServeConfig struct {
	StartFunc func(ctx context.Context) error
}

// DemoConfig configures the Kabuki demo server command.
type DemoConfig struct {
	StartFunc func(ctx context.Context, port int, speed float64) error
}

// ProfileConfig configures the Ouroboros profile command.
type ProfileConfig struct {
	RunFunc     func(ctx context.Context, model string) error
	ReportFunc  func(ctx context.Context) error
	CompareFunc func(ctx context.Context, a, b string) error
}

// CLIBuilder assembles a consumer CLI from Origami-provided commands.
type CLIBuilder struct {
	name        string
	description string
	version     string

	analyzeFn   AnalyzeFunc
	dataset     DatasetStore
	calibrate   CalibrateRunner
	pipelines   []string
	consume     *string
	serve       *ServeConfig
	demo        *DemoConfig
	profile     *ProfileConfig
	extra       []*cobra.Command
	observers   []func() // observability setup hooks
}

// NewCLI creates a CLI builder for the named tool.
func NewCLI(name, description string) *CLIBuilder {
	return &CLIBuilder{name: name, description: description}
}

func (b *CLIBuilder) WithVersion(v string) *CLIBuilder {
	b.version = v
	return b
}

func (b *CLIBuilder) WithAnalyze(fn AnalyzeFunc) *CLIBuilder {
	b.analyzeFn = fn
	return b
}

func (b *CLIBuilder) WithDataset(store DatasetStore) *CLIBuilder {
	b.dataset = store
	return b
}

func (b *CLIBuilder) WithCalibrate(runner CalibrateRunner) *CLIBuilder {
	b.calibrate = runner
	return b
}

func (b *CLIBuilder) WithPipeline(defs ...string) *CLIBuilder {
	b.pipelines = append(b.pipelines, defs...)
	return b
}

func (b *CLIBuilder) WithConsume(pipeline string) *CLIBuilder {
	b.consume = &pipeline
	return b
}

func (b *CLIBuilder) WithServe(cfg ServeConfig) *CLIBuilder {
	b.serve = &cfg
	return b
}

func (b *CLIBuilder) WithDemo(cfg DemoConfig) *CLIBuilder {
	b.demo = &cfg
	return b
}

func (b *CLIBuilder) WithProfile(cfg ProfileConfig) *CLIBuilder {
	b.profile = &cfg
	return b
}

func (b *CLIBuilder) WithExtraCommand(cmd *cobra.Command) *CLIBuilder {
	b.extra = append(b.extra, cmd)
	return b
}

// CLI is the assembled command tree ready for execution.
type CLI struct {
	root *cobra.Command
}

// Build assembles the Cobra command tree. Returns an error if required
// configuration is missing (analyze is required).
func (b *CLIBuilder) Build() (*CLI, error) {
	if b.analyzeFn == nil {
		return nil, fmt.Errorf("WithAnalyze is required")
	}

	root := &cobra.Command{
		Use:   b.name,
		Short: b.description,
	}

	root.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	root.PersistentFlags().Bool("debug", false, "debug-level output")
	root.PersistentFlags().StringP("output", "o", "table", "output format (json, table, markdown)")
	root.PersistentFlags().String("config", "", "config file path")

	if b.version != "" {
		root.Version = b.version
	}

	root.AddCommand(b.buildAnalyze())

	if b.dataset != nil {
		root.AddCommand(b.buildDataset())
	}
	if b.calibrate != nil {
		root.AddCommand(b.buildCalibrate())
	}
	if len(b.pipelines) > 0 {
		root.AddCommand(b.buildPipeline())
	}
	if b.consume != nil {
		root.AddCommand(b.buildConsume())
	}
	if b.serve != nil {
		root.AddCommand(b.buildServe())
	}
	if b.demo != nil {
		root.AddCommand(b.buildDemo())
	}
	if b.profile != nil {
		root.AddCommand(b.buildProfile())
	}
	for _, cmd := range b.extra {
		root.AddCommand(cmd)
	}

	return &CLI{root: root}, nil
}

// Execute runs the CLI with os.Args. The standard entry point.
func (c *CLI) Execute() error {
	return c.root.Execute()
}

// Root returns the underlying Cobra command for testing or customization.
func (c *CLI) Root() *cobra.Command {
	return c.root
}

func (b *CLIBuilder) buildAnalyze() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze [args...]",
		Short: "Run the domain analysis function",
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.analyzeFn(cmd.Context(), args)
		},
	}
}

func (b *CLIBuilder) buildDataset() *cobra.Command {
	ds := &cobra.Command{
		Use:   "dataset",
		Short: "Manage datasets",
	}

	ds.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available datasets",
		RunE: func(_ *cobra.Command, _ []string) error {
			items, err := b.dataset.List()
			if err != nil {
				return err
			}
			for _, d := range items {
				fmt.Printf("%-20s %d cases  [%s]\n", d.Name, d.CaseCount, d.Status)
			}
			return nil
		},
	})

	ds.AddCommand(&cobra.Command{
		Use:   "status <name>",
		Short: "Show dataset status",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := b.dataset.Status(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Dataset: %s\nCases: %d\nReviewed: %d\nPromoted: %d\n",
				st.Name, st.CaseCount, st.Reviewed, st.Promoted)
			return nil
		},
	})

	importCmd := &cobra.Command{
		Use:   "import <path>",
		Short: "Import a dataset from path",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return b.dataset.Import(args[0])
		},
	}
	ds.AddCommand(importCmd)

	ds.AddCommand(&cobra.Command{
		Use:   "export <path>",
		Short: "Export dataset to path",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return b.dataset.Export(args[0])
		},
	})

	ds.AddCommand(&cobra.Command{
		Use:   "review",
		Short: "List candidates for review",
		RunE: func(_ *cobra.Command, _ []string) error {
			candidates, err := b.dataset.ListCandidates()
			if err != nil {
				return err
			}
			for _, c := range candidates {
				fmt.Printf("%-30s  %s\n", c.CaseID, c.Source)
			}
			return nil
		},
	})

	ds.AddCommand(&cobra.Command{
		Use:   "promote <case-id>",
		Short: "Promote a case to the dataset",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return b.dataset.Promote(args[0])
		},
	})

	return ds
}

func (b *CLIBuilder) buildCalibrate() *cobra.Command {
	cal := &cobra.Command{
		Use:   "calibrate",
		Short: "Run scorecard evaluation",
	}

	cal.AddCommand(&cobra.Command{
		Use:   "run <scenario>",
		Short: "Run calibration against a scenario",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.calibrate.Run(cmd.Context(), args[0])
		},
	})

	cal.AddCommand(&cobra.Command{
		Use:   "report <scenario>",
		Short: "Show calibration report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.calibrate.Report(cmd.Context(), args[0])
		},
	})

	cal.AddCommand(&cobra.Command{
		Use:   "compare <a> <b>",
		Short: "Compare two calibration runs",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.calibrate.Compare(cmd.Context(), args[0], args[1])
		},
	})

	return cal
}

func (b *CLIBuilder) buildPipeline() *cobra.Command {
	pl := &cobra.Command{
		Use:   "pipeline",
		Short: "Pipeline operations",
	}

	pl.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List registered pipelines",
		RunE: func(_ *cobra.Command, _ []string) error {
			for _, p := range b.pipelines {
				fmt.Println(p)
			}
			return nil
		},
	})

	pl.AddCommand(&cobra.Command{
		Use:   "validate <pipeline>",
		Short: "Validate a pipeline YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			fmt.Printf("Validating %s... (not yet implemented)\n", args[0])
			return nil
		},
	})

	pl.AddCommand(&cobra.Command{
		Use:   "render <pipeline>",
		Short: "Render pipeline as DOT graph",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			fmt.Printf("Rendering %s... (not yet implemented)\n", args[0])
			return nil
		},
	})

	replayCmd := &cobra.Command{
		Use:   "replay <recording.jsonl>",
		Short: "Replay a pipeline recording via Kami",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			fmt.Printf("Replaying %s... (not yet implemented)\n", args[0])
			return nil
		},
	}
	replayCmd.Flags().Float64("speed", 1.0, "replay speed multiplier")
	replayCmd.Flags().Int("port", 3000, "Kami server port")
	pl.AddCommand(replayCmd)

	return pl
}

func (b *CLIBuilder) buildConsume() *cobra.Command {
	consume := &cobra.Command{
		Use:   "consume",
		Short: "Ingest data from sources",
	}
	consume.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run ingestion pipeline",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Printf("Running consume pipeline %s...\n", *b.consume)
			return nil
		},
	})
	consume.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show last ingestion status",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("Ingestion status: not yet implemented")
			return nil
		},
	})
	return consume
}

func (b *CLIBuilder) buildServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run as MCP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return b.serve.StartFunc(cmd.Context())
		},
	}
}

func (b *CLIBuilder) buildDemo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Start Kabuki presentation server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			port, _ := cmd.Flags().GetInt("port")
			speed, _ := cmd.Flags().GetFloat64("speed")
			return b.demo.StartFunc(cmd.Context(), port, speed)
		},
	}
	cmd.Flags().Int("port", 3000, "server port")
	cmd.Flags().Float64("speed", 1.0, "presentation speed")
	return cmd
}

func (b *CLIBuilder) buildProfile() *cobra.Command {
	prof := &cobra.Command{
		Use:   "profile",
		Short: "Ouroboros model discovery",
	}
	prof.AddCommand(&cobra.Command{
		Use:   "run <model>",
		Short: "Run discovery probes on a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.profile.RunFunc(cmd.Context(), args[0])
		},
	})
	prof.AddCommand(&cobra.Command{
		Use:   "report",
		Short: "Show discovery report",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return b.profile.ReportFunc(cmd.Context())
		},
	})
	prof.AddCommand(&cobra.Command{
		Use:   "compare <a> <b>",
		Short: "Compare two model profiles",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return b.profile.CompareFunc(cmd.Context(), args[0], args[1])
		},
	})
	return prof
}
