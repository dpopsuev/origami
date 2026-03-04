package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dpopsuev/origami/schematics/rca"
	"github.com/dpopsuev/origami/schematics/rca/store"
	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/knowledge"
)

var analyzeFlags struct {
	launch        string
	workspacePath string
	artifactPath  string
	dbPath        string
	backendName   string
	dispatchMode  string
	promptDir     string
	rpBase        string
	rpKeyPath     string
	rpProject     string
	report        bool
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [launch-id]",
	Short: "Run evidence-based RCA on a ReportPortal launch",
	Long: `Analyze failures from a ReportPortal launch and produce an RCA artifact
with defect classifications and confidence scores.

Usage:
  asterisk analyze 33195                    # Launch ID as positional arg
  asterisk analyze --launch=33195           # Launch ID as flag
  asterisk analyze path/to/envelope.json    # Local envelope file

The RP base URL is read from the ASTERISK_RP_URL environment variable,
or can be set with --rp-base-url. If neither is set, the tool will
prompt you to configure it.

The RP API token is read from .rp-api-key (first line). If the file
does not exist, the tool will show you how to get and save the token.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	f := analyzeCmd.Flags()
	f.StringVar(&analyzeFlags.launch, "launch", "", "Path to envelope JSON or launch ID")
	f.StringVar(&analyzeFlags.workspacePath, "workspace", "", "Path to context workspace file (YAML/JSON)")
	f.StringVarP(&analyzeFlags.artifactPath, "output", "o", "", "Output artifact path (default: .asterisk/output/rca-<launch>.json)")
	f.StringVar(&analyzeFlags.dbPath, "db", store.DefaultDBPath, "Store DB path")
	f.StringVar(&analyzeFlags.backendName, "backend", "basic", "Backend: basic (heuristic) or llm (AI via LLM agent)")
	f.StringVar(&analyzeFlags.dispatchMode, "dispatch", "file", "Dispatch mode for cursor backend (stdin, file)")
	f.StringVar(&analyzeFlags.promptDir, "prompt-dir", "", "Prompt template directory (default: embedded prompts)")
	f.StringVar(&analyzeFlags.rpBase, "rp-base-url", "", "RP base URL (default: $ASTERISK_RP_URL)")
	f.StringVar(&analyzeFlags.rpKeyPath, "rp-api-key", ".rp-api-key", "Path to RP API key file")
	f.StringVar(&analyzeFlags.rpProject, "rp-project", "", "RP project name (default: $ASTERISK_RP_PROJECT)")
	f.BoolVar(&analyzeFlags.report, "report", false, "Write a human-readable Markdown report (.md) alongside the JSON artifact")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	launch := analyzeFlags.launch
	if launch == "" && len(args) > 0 {
		launch = args[0]
	}
	if launch == "" {
		return fmt.Errorf("launch ID or envelope path is required\n\nUsage: asterisk analyze <launch-id>\n       asterisk analyze path/to/envelope.json")
	}

	rpBase := analyzeFlags.rpBase
	if rpBase == "" {
		rpBase = os.Getenv("ASTERISK_RP_URL")
	}

	if _, err := strconv.Atoi(launch); err == nil && rpBase == "" {
		return fmt.Errorf("RP base URL is required when using a launch ID\n\nSet it via environment variable:\n  export ASTERISK_RP_URL=https://your-rp-instance.example.com\n\nOr use the --rp-base-url flag:\n  asterisk analyze %s --rp-base-url https://your-rp-instance.example.com", launch)
	}

	if rpBase != "" {
		if err := checkTokenFileViaOption(analyzeFlags.rpKeyPath); err != nil {
			return err
		}
	}

	artifactPath := analyzeFlags.artifactPath
	if artifactPath == "" {
		safeName := launch
		if id, err := strconv.Atoi(launch); err == nil {
			safeName = strconv.Itoa(id)
		} else {
			safeName = filepath.Base(launch)
		}
		outputDir := filepath.Join(".asterisk", "output")
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		artifactPath = filepath.Join(outputDir, fmt.Sprintf("rca-%s.json", safeName))
	}

	rpProject := resolveRPProject(analyzeFlags.rpProject)
	if rpBase != "" && rpProject == "" {
		return fmt.Errorf("RP project name is required when using RP API\n\nSet it via environment variable:\n  export ASTERISK_RP_PROJECT=your-project-name\n\nOr use the --rp-project flag:\n  asterisk analyze %s --rp-project your-project-name", launch)
	}

	var source rca.SourceReader
	if rpBase != "" && cfg.readerFactory != nil {
		var err error
		source, err = cfg.readerFactory(rpBase, analyzeFlags.rpKeyPath, rpProject)
		if err != nil {
			return fmt.Errorf("create source reader: %w", err)
		}
	}
	env := loadEnvelopeForAnalyze(launch, analyzeFlags.dbPath, source)
	if env == nil {
		return fmt.Errorf("could not load envelope for launch %q", launch)
	}
	if len(env.FailureList) == 0 {
		return fmt.Errorf("envelope has no failures")
	}

	st, err := openStore(analyzeFlags.dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	var catalog *knowledge.KnowledgeSourceCatalog
	var repoNames []string
	if analyzeFlags.workspacePath != "" {
		cat, err := knowledge.LoadFromPath(analyzeFlags.workspacePath)
		if err != nil {
			return fmt.Errorf("load catalog: %w", err)
		}
		catalog = cat
		for _, s := range catalog.Sources {
			repoNames = append(repoNames, s.Name)
		}
	} else {
		repoNames = defaultWorkspaceRepos()
	}

	suiteID, cases := createAnalysisScaffolding(st, env)

	cfg := rca.AnalysisConfig{
		Thresholds: rca.DefaultThresholds(),
		Envelope:   env,
		Catalog:    catalog,
	}
	switch analyzeFlags.backendName {
	case "basic":
		cfg.Components = []*framework.Component{rca.HeuristicComponent(st, repoNames)}
	case "llm":
		dispatcher, err := buildDispatcher(DispatchOpts{Mode: analyzeFlags.dispatchMode})
		if err != nil {
			return err
		}
		basePath := filepath.Join(".asterisk", "analyze")
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return fmt.Errorf("create analyze dir: %w", err)
		}
		t := rca.NewRCATransformer(dispatcher, resolvePromptFS(analyzeFlags.promptDir),
			rca.WithRCABasePath(basePath),
		)
		cfg.Components = []*framework.Component{rca.TransformerComponent(t)}
		cfg.BasePath = basePath
	default:
		return fmt.Errorf("unknown backend: %s (supported: basic, llm)", analyzeFlags.backendName)
	}
	report, err := rca.RunAnalysis(st, cases, suiteID, cfg)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	report.SourceName = env.Name

	for i := range report.CaseResults {
		if i < len(env.FailureList) {
			f := env.FailureList[i]
			if f.Tags != nil {
				report.CaseResults[i].SourceIssueType = f.Tags["rp.issue_type"]
				report.CaseResults[i].SourceAutoAnalyzed = f.Tags["rp.auto_analyzed"] == "true"
			}
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(artifactPath, data, 0600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), rca.FormatAnalysisReport(report))
	fmt.Fprintf(cmd.OutOrStdout(), "\nReport written to: %s\n", artifactPath)

	if analyzeFlags.report {
		mdPath := strings.TrimSuffix(artifactPath, ".json") + ".md"
		mdContent, renderErr := rca.RenderAnalysisReport(report, time.Now())
		if renderErr != nil {
			return fmt.Errorf("render RCA report: %w", renderErr)
		}
		if err := os.WriteFile(mdPath, []byte(mdContent), 0600); err != nil {
			return fmt.Errorf("write report markdown: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Human-readable report: %s\n", mdPath)
	}

	return nil
}
