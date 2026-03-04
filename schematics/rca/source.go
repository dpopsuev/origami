package rca

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dpopsuev/origami/schematics/rca/rcatype"
)

// SourceFactory creates a SourceAdapter from connection parameters.
// Products inject a factory so the schematic never imports connector packages.
type SourceFactory func(baseURL, apiKeyPath, project string) (SourceAdapter, error)

// SourceAdapter abstracts the external test failure tracker. Schematics
// declare this as a required socket; concrete connectors (e.g. ReportPortal)
// satisfy it and are injected at build time via functional options.
type SourceAdapter interface {
	// FetchEnvelope retrieves test failure data for the given source launch ID.
	FetchEnvelope(launchID int) (*rcatype.Envelope, error)

	// EnvelopeFetcher returns an rcatype.EnvelopeFetcher for batch operations
	// like calibration case resolution.
	EnvelopeFetcher() rcatype.EnvelopeFetcher

	// CurrentUser returns the identity of the authenticated user.
	CurrentUser() string
}

// PusherFactory creates a DefectPusher from connection parameters.
type PusherFactory func(baseURL, apiKeyPath, project, submittedBy string) (DefectPusher, error)

// DefectPusher pushes RCA results back to the external tracker.
type DefectPusher interface {
	Push(artifactPath, jiraTicketID, jiraLink string) (*PushedRecord, error)
}

// PushedRecord captures the result of a defect push operation.
type PushedRecord struct {
	LaunchID   string
	DefectType string
}

// DefaultDefectPusher reads an RCA artifact and extracts the defect type
// locally without contacting any remote API.
type DefaultDefectPusher struct{}

func (DefaultDefectPusher) Push(artifactPath, jiraTicketID, jiraLink string) (*PushedRecord, error) {
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

// LaunchInfo summarizes a CI launch for the ingestion circuit.
type LaunchInfo struct {
	ID          int       `json:"id"`
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	Number      int       `json:"number"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	FailedCount int       `json:"failed_count"`
}

// FailureInfo represents a parsed test failure from a CI launch.
type FailureInfo struct {
	LaunchID     int    `json:"launch_id"`
	LaunchName   string `json:"launch_name"`
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
	return fmt.Sprintf("%s:%d:%d", project, f.LaunchID, f.ItemID)
}

// LaunchFetcher abstracts the external tracker's launch listing API.
type LaunchFetcher interface {
	FetchLaunches(project string, since time.Time) ([]LaunchInfo, error)
	FetchFailures(launchID int) ([]FailureInfo, error)
}

// LaunchFetcherFactory creates a LaunchFetcher from connection parameters.
type LaunchFetcherFactory func(baseURL, apiKeyPath, project string) (LaunchFetcher, error)
