package rca

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dpopsuev/origami/schematics/rca/rcatype"
)

// SourceReaderFactory creates a SourceReader from connection parameters.
// Products inject a factory so the schematic never imports connector packages.
type SourceReaderFactory func(baseURL, apiKeyPath, project string) (SourceReader, error)

// SourceReader reads test failure data from an external tracker. Schematics
// declare this as a required socket; concrete connectors (e.g. ReportPortal)
// satisfy it and are injected at build time via functional options.
type SourceReader interface {
	// FetchEnvelope retrieves test failure data for the given run ID.
	FetchEnvelope(launchID int) (*rcatype.Envelope, error)

	// EnvelopeFetcher returns an rcatype.EnvelopeFetcher for batch operations
	// like calibration case resolution.
	EnvelopeFetcher() rcatype.EnvelopeFetcher

	// CurrentUser returns the identity of the authenticated user.
	CurrentUser() string
}

// DefectWriterFactory creates a DefectWriter from connection parameters.
type DefectWriterFactory func(baseURL, apiKeyPath, project, submittedBy string) (DefectWriter, error)

// DefectWriter writes RCA results back to an external system.
type DefectWriter interface {
	Push(artifactPath, jiraTicketID, jiraLink string) (*PushedRecord, error)
}

// PushedRecord captures the result of a defect write operation.
type PushedRecord struct {
	LaunchID   string
	DefectType string
}

// DefaultDefectWriter reads an RCA artifact and extracts the defect type
// locally without contacting any remote API.
type DefaultDefectWriter struct{}

func (DefaultDefectWriter) Push(artifactPath, jiraTicketID, jiraLink string) (*PushedRecord, error) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, err
	}
	var a struct {
		LaunchID   string `json:"launch_id"`
		DefectType string `json:"defect_type"`
	}
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &PushedRecord{LaunchID: a.LaunchID, DefectType: a.DefectType}, nil
}

// TokenChecker validates the presence and permissions of a token file.
type TokenChecker func(path string) error

// RunInfo summarizes a CI run for the ingestion circuit.
type RunInfo struct {
	ID          int       `json:"id"`
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	Number      int       `json:"number"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	FailedCount int       `json:"failed_count"`
}

// FailureInfo represents a parsed test failure from a CI run.
type FailureInfo struct {
	RunID        int    `json:"run_id"`
	RunName      string `json:"run_name"`
	ItemID       int    `json:"item_id"`
	ItemUUID     string `json:"item_uuid"`
	TestName     string `json:"test_name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	IssueType    string `json:"issue_type,omitempty"`
	AutoAnalyzed bool   `json:"auto_analyzed,omitempty"`
}

// DedupKey generates the deduplication key for a failure.
func (f *FailureInfo) DedupKey(project string) string {
	return fmt.Sprintf("%s:%d:%d", project, f.RunID, f.ItemID)
}

// RunDiscoverer discovers available CI runs and their failures.
type RunDiscoverer interface {
	DiscoverRuns(project string, since time.Time) ([]RunInfo, error)
	FetchFailures(runID int) ([]FailureInfo, error)
}

// RunDiscovererFactory creates a RunDiscoverer from connection parameters.
type RunDiscovererFactory func(baseURL, apiKeyPath, project string) (RunDiscoverer, error)
