package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpopsuev/origami/knowledge"
	"github.com/dpopsuev/origami/schematics/rca"
	"github.com/dpopsuev/origami/schematics/rca/store"
)

var cursorFlags struct {
	launch    string
	sources   string
	itemID    string
	promptDir string
	dbPath    string
}

var cursorCmd = &cobra.Command{
	Use:   "cursor",
	Short: "Run interactive Cursor-based RCA for a single failure",
	Long: `Generate a prompt for the next circuit step and print it.
Paste the prompt into Cursor, save the artifact, then run again to advance.`,
	RunE: runCursor,
}

func init() {
	f := cursorCmd.Flags()
	f.StringVar(&cursorFlags.launch, "launch", "", "Path to envelope JSON or launch ID (required)")
	f.StringVar(&cursorFlags.sources, "sources", "", "Comma-separated source pack names (e.g. ptp,nrop)")
	f.StringVar(&cursorFlags.itemID, "case-id", "", "Failure (test item) source ID; default first from envelope")
	f.StringVar(&cursorFlags.promptDir, "prompt-dir", "", "Prompt template directory (default: embedded prompts)")
	f.StringVar(&cursorFlags.dbPath, "db", store.DefaultDBPath, "Store DB path")

	_ = cursorCmd.MarkFlagRequired("launch")
}

func runCursor(cmd *cobra.Command, _ []string) error {
	env, rpLaunchID := loadEnvelopeForCursor(cursorFlags.launch, cursorFlags.dbPath)
	if env == nil {
		return fmt.Errorf("could not load envelope for launch %q", cursorFlags.launch)
	}
	if len(env.FailureList) == 0 {
		return fmt.Errorf("envelope has no failures")
	}

	item := env.FailureList[0]
	for _, f := range env.FailureList {
		if cursorFlags.itemID == "" || f.ID == cursorFlags.itemID {
			item = f
			break
		}
	}
	if cursorFlags.itemID != "" && item.ID != cursorFlags.itemID {
		return fmt.Errorf("case-id %s not in envelope", cursorFlags.itemID)
	}

	st, err := openStore(cursorFlags.dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	caseData := ensureCaseInStore(st, env, rpLaunchID, item)

	var catalog *knowledge.KnowledgeSourceCatalog
	if cursorFlags.sources != "" {
		sourceNames := strings.Split(cursorFlags.sources, ",")
		_, cat, err := loadSourcePacks(sourceNames, nil)
		if err != nil {
			return err
		}
		catalog = cat
	}

	suiteID := int64(1)
	caseDir, err := rca.EnsureCaseDir(rca.DefaultBasePath, suiteID, caseData.ID)
	if err != nil {
		return fmt.Errorf("ensure case dir: %w", err)
	}

	result, err := rca.RunHITLStep(context.Background(), rca.HITLConfig{
		Store:    st,
		CaseData: caseData,
		Envelope: env,
		Catalog:  catalog,
		PromptFS: resolvePromptFS(cursorFlags.promptDir),
		CaseDir:  caseDir,
	})
	if err != nil {
		return fmt.Errorf("orchestrate: %w", err)
	}

	out := cmd.OutOrStdout()
	if result.IsDone {
		fmt.Fprintf(out, "Circuit complete for case #%d. %s\n", caseData.ID, result.Explanation)
		return nil
	}

	fmt.Fprintf(out, "Step: %s\n", vocabNameWithCode(result.CurrentStep))
	fmt.Fprintf(out, "Prompt: %s\n", result.PromptPath)
	fmt.Fprintf(out, "\nPaste the prompt into Cursor, then save the artifact to the case directory.\n")
	fmt.Fprintf(out, "Run 'asterisk save' to ingest the artifact and advance.\n")
	return nil
}
