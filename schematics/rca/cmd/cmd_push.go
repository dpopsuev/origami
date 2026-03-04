package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/dpopsuev/origami/schematics/rca"
)

var pushFlags struct {
	artifactPath string
	rpBase       string
	rpKeyPath    string
	rpProject    string
	submittedBy  string
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push an RCA artifact to ReportPortal as a defect update",
	RunE:  runPush,
}

func init() {
	f := pushCmd.Flags()
	f.StringVarP(&pushFlags.artifactPath, "file", "f", "", "Artifact file path (required)")
	f.StringVar(&pushFlags.rpBase, "rp-base-url", "", "RP base URL (optional)")
	f.StringVar(&pushFlags.rpKeyPath, "rp-api-key", ".rp-api-key", "Path to RP API key file")
	f.StringVar(&pushFlags.rpProject, "rp-project", "", "RP project name (default: $ASTERISK_RP_PROJECT)")
	f.StringVar(&pushFlags.submittedBy, "submitted-by", "", "Attribution name (default: resolved from RP API token, $ASTERISK_USER, or system username)")

	_ = pushCmd.MarkFlagRequired("file")
}

func runPush(cmd *cobra.Command, _ []string) error {
	var pusher rca.DefectPusher = rca.DefaultDefectPusher{}
	if pushFlags.rpBase != "" {
		rpProject := resolveRPProject(pushFlags.rpProject)
		if rpProject == "" {
			return fmt.Errorf("RP project name is required when using RP API\n\nSet it via environment variable:\n  export ASTERISK_RP_PROJECT=your-project-name\n\nOr use the --rp-project flag:\n  asterisk push -f artifact.json --rp-base-url ... --rp-project your-project-name")
		}
		if cfg.pusherFactory == nil {
			return fmt.Errorf("no defect pusher configured (pusher factory not injected)")
		}
		submitter := resolveSubmitter()
		p, err := cfg.pusherFactory(pushFlags.rpBase, pushFlags.rpKeyPath, rpProject, submitter)
		if err != nil {
			return fmt.Errorf("create defect pusher: %w", err)
		}
		pusher = p
	}
	rec, err := pusher.Push(pushFlags.artifactPath, "", "")
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	if rec != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Pushed: launch=%s defect_type=%s\n", rec.LaunchID, vocabNameWithCode(rec.DefectType))
	}
	return nil
}

func resolveSubmitter() string {
	if pushFlags.submittedBy != "" {
		return pushFlags.submittedBy
	}
	if v := os.Getenv("ASTERISK_USER"); v != "" {
		return v
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return ""
}
