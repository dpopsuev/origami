package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dpopsuev/origami/dispatch"

	"github.com/dpopsuev/origami/schematics/rca"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/store"
)

// DispatchOpts collects the parameters needed to construct a dispatcher.
type DispatchOpts struct {
	Mode      string
	Logger    *slog.Logger
	SuiteDir  string // batch-file only
	BatchSize int    // batch-file only
}

// buildDispatcher creates a dispatch.Dispatcher from declarative options.
// Supported modes: stdin, file, batch-file.
func buildDispatcher(opts DispatchOpts) (dispatch.Dispatcher, error) {
	switch opts.Mode {
	case "stdin":
		return dispatch.NewStdinDispatcherWithTemplate(asteriskStdinTemplate()), nil
	case "file":
		cfg := dispatch.DefaultFileDispatcherConfig()
		cfg.Logger = opts.Logger
		return dispatch.NewFileDispatcher(cfg), nil
	case "batch-file":
		suiteDir := opts.SuiteDir
		if suiteDir == "" {
			suiteDir = filepath.Join(".asterisk", "calibrate", "batch")
		}
		batchSize := opts.BatchSize
		if batchSize <= 0 {
			batchSize = 4
		}
		cfg := dispatch.BatchFileDispatcherConfig{
			FileConfig: dispatch.FileDispatcherConfig{Logger: opts.Logger},
			SuiteDir:   suiteDir,
			BatchSize:  batchSize,
			Logger:     opts.Logger,
		}
		return dispatch.NewBatchFileDispatcher(cfg), nil
	default:
		return nil, fmt.Errorf("unknown dispatch mode: %s (available: stdin, file, batch-file)", opts.Mode)
	}
}

func asteriskStdinTemplate() dispatch.StdinTemplate {
	return dispatch.StdinTemplate{
		Instructions: []string{
			"1. Open the prompt file and paste it into Cursor",
			"2. Save Cursor's JSON response to the artifact path above",
			"3. Press Enter to continue",
		},
	}
}

func checkTokenFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("RP API token file not found: %s\n\n"+
			"To get your RP API token:\n"+
			"  1. Log in to your ReportPortal instance\n"+
			"  2. Go to User Profile (top-right icon)\n"+
			"  3. Copy the API token (UUID format)\n"+
			"  4. Save it:  echo '<YOUR_TOKEN>' > .rp-api-key && chmod 600 .rp-api-key\n", path)
	}
	if err != nil {
		return fmt.Errorf("check token file: %w", err)
	}
	if perm := info.Mode().Perm(); perm&0044 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %s is readable by group/others (mode %04o). Run: chmod 600 %s\n", path, perm, path)
	}
	return nil
}

func defaultWorkspaceRepos() []string {
	return []string{
		"ptp-operator",
		"linuxptp-daemon",
		"linuxptp-daemon-v2",
		"cloud-event-proxy",
		"ptp-operator-must-gather",
		"cluster-etcd-operator",
	}
}

// resolvePromptFS returns an fs.FS for prompt templates. When dir is non-empty
// the prompts are read from that disk directory (wrapped via os.DirFS); when
// empty the compiled-in DefaultPromptFS is used.
func resolvePromptFS(dir string) fs.FS {
	if dir != "" {
		return os.DirFS(dir)
	}
	return rca.DefaultPromptFS
}

// resolveRPProject returns the RP project name from the given flag value,
// falling back to $ASTERISK_RP_PROJECT. Returns "" if neither is set.
func resolveRPProject(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("ASTERISK_RP_PROJECT")
}

// loadEnvelopeForAnalyze resolves the envelope from a file path or launch ID.
// When a SourceReader is available and the envelope is not cached, it fetches
// from the remote tracker and saves to the store.
func loadEnvelopeForAnalyze(launch, dbPath string, source rca.SourceReader) *rcatype.Envelope {
	if _, err := os.Stat(launch); err == nil {
		data, err := os.ReadFile(launch)
		if err != nil {
			return nil
		}
		var env rcatype.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil
		}
		return &env
	}

	launchID, err := strconv.Atoi(launch)
	if err != nil || launchID <= 0 {
		return nil
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil
	}
	defer st.Close()

	env, _ := st.GetEnvelope(launchID)
	if env == nil && source != nil {
		fetched, err := source.FetchEnvelope(launchID)
		if err != nil {
			return nil
		}
		_ = st.SaveEnvelope(launchID, fetched)
		env = fetched
	}
	return env
}

// loadEnvelopeForCursor resolves the envelope from a file path or launch ID for cursor mode.
func loadEnvelopeForCursor(launch string, dbPath string) (*rcatype.Envelope, int) {
	if _, err := os.Stat(launch); err == nil {
		data, err := os.ReadFile(launch)
		if err != nil {
			return nil, 0
		}
		var env rcatype.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, 0
		}
		launchID, _ := strconv.Atoi(env.RunID)
		if launchID == 0 {
			launchID = 1
		}
		return &env, launchID
	}
	launchID, err := strconv.Atoi(launch)
	if err != nil || launchID <= 0 {
		return nil, 0
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, 0
	}
	defer st.Close()
	env, _ := st.GetEnvelope(launchID)
	if env == nil {
		return nil, 0
	}
	return env, launchID
}

// createAnalysisScaffolding creates v2 store entities for all failures in the envelope.
func createAnalysisScaffolding(st store.Store, env *rcatype.Envelope) (int64, []*store.Case) {
	rpLaunchID, _ := strconv.Atoi(env.RunID)

	suiteID, _ := st.CreateSuite(&store.InvestigationSuite{
		Name:        fmt.Sprintf("Analysis %s", env.Name),
		Description: fmt.Sprintf("Automated analysis for launch %s", env.RunID),
		Status:      "active",
	})

	vID, _ := st.CreateVersion(&store.Version{Label: "unknown"})
	if vID == 0 {
		v, _ := st.GetVersionByLabel("unknown")
		if v != nil {
			vID = v.ID
		}
	}

	pID, _ := st.CreateCircuit(&store.Circuit{
		SuiteID:    suiteID,
		VersionID:  vID,
		Name:       env.Name,
		SourceLaunchID: rpLaunchID,
		Status:     "complete",
	})

	lID, _ := st.CreateLaunch(&store.Launch{
		CircuitID: pID,
		SourceLaunchID: rpLaunchID,
		Name:       env.Name,
		Status:     "complete",
	})

	jID, _ := st.CreateJob(&store.Job{
		LaunchID: lID,
		Name:     env.Name,
		Status:   "complete",
	})

	var cases []*store.Case
	for _, f := range env.FailureList {
		caseID, _ := st.CreateCase(&store.Case{
			JobID:    jID,
			LaunchID: lID,
			SourceItemID: f.ID,
			Name:     f.Name,
			Status:   "open",
		})
		c, _ := st.GetCase(caseID)
		if c != nil {
			cases = append(cases, c)
		}
	}

	return suiteID, cases
}

// ensureCaseInStore finds or creates the full v2 scaffolding for a failure item.
func ensureCaseInStore(st store.Store, env *rcatype.Envelope, rpLaunchID int, item rcatype.FailureItem) *store.Case {
	suites, _ := st.ListSuites()
	for _, suite := range suites {
		if suite.Status != "open" {
			continue
		}
		circuits, _ := st.ListCircuitsBySuite(suite.ID)
		for _, p := range circuits {
			launches, _ := st.ListLaunchesByCircuit(p.ID)
			for _, l := range launches {
				jobs, _ := st.ListJobsByLaunch(l.ID)
				for _, j := range jobs {
					cases, _ := st.ListCasesByJob(j.ID)
					for _, c := range cases {
						if c.SourceItemID == item.ID {
							return c
						}
					}
				}
			}
		}
	}

	suiteID, _ := st.CreateSuite(&store.InvestigationSuite{
		Name:        fmt.Sprintf("Investigation %s", env.Name),
		Description: fmt.Sprintf("Auto-created for launch %s", env.RunID),
	})

	vID, _ := st.CreateVersion(&store.Version{Label: "unknown"})
	if vID == 0 {
		v, _ := st.GetVersionByLabel("unknown")
		if v != nil {
			vID = v.ID
		}
	}

	pID, _ := st.CreateCircuit(&store.Circuit{
		SuiteID:    suiteID,
		VersionID:  vID,
		Name:       env.Name,
		SourceLaunchID: rpLaunchID,
	})

	lID, _ := st.CreateLaunch(&store.Launch{
		CircuitID: pID,
		SourceLaunchID: rpLaunchID,
		Name:       env.Name,
	})

	jID, _ := st.CreateJob(&store.Job{
		LaunchID: lID,
		SourceItemID: item.ID,
		Name:     item.Name,
	})

	caseID, _ := st.CreateCase(&store.Case{
		JobID:    jID,
		LaunchID: lID,
		SourceItemID: item.ID,
		Name:     item.Name,
		Status:   "open",
	})

	caseData, _ := st.GetCase(caseID)
	if caseData == nil {
		fmt.Fprintf(os.Stderr, "failed to create case in store\n")
		os.Exit(1)
	}
	return caseData
}
