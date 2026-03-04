package rp

import (
	"context"
	"time"

	"github.com/dpopsuev/origami/schematics/rca"
)

var _ rca.LaunchFetcher = (*RPLaunchFetcher)(nil)

// RPLaunchFetcher implements rca.LaunchFetcher for ReportPortal.
type RPLaunchFetcher struct {
	client  *Client
	project string
}

// NewLaunchFetcher creates a LaunchFetcher backed by a ReportPortal API client.
func NewLaunchFetcher(baseURL, apiKeyPath, project string) (rca.LaunchFetcher, error) {
	key, err := ReadAPIKey(apiKeyPath)
	if err != nil {
		return nil, err
	}
	client, err := New(baseURL, key, WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	return &RPLaunchFetcher{client: client, project: project}, nil
}

func (f *RPLaunchFetcher) FetchLaunches(project string, since time.Time) ([]rca.LaunchInfo, error) {
	ctx := context.Background()
	paged, err := f.client.Project(project).Launches().List(ctx,
		WithPageSize(100),
		WithSort("startTime,desc"),
	)
	if err != nil {
		return nil, err
	}

	var launches []rca.LaunchInfo
	for _, l := range paged.Content {
		var startTime time.Time
		if l.StartTime != nil {
			startTime = l.StartTime.Time()
		}
		if !since.IsZero() && startTime.Before(since) {
			continue
		}
		failed := 0
		if l.Statistics != nil {
			if execs, ok := l.Statistics.Executions["failed"]; ok {
				failed = execs
			}
		}
		launches = append(launches, rca.LaunchInfo{
			ID:          l.ID,
			UUID:        l.UUID,
			Name:        l.Name,
			Number:      l.Number,
			Status:      l.Status,
			StartTime:   startTime,
			FailedCount: failed,
		})
	}
	return launches, nil
}

func (f *RPLaunchFetcher) FetchFailures(launchID int) ([]rca.FailureInfo, error) {
	ctx := context.Background()
	items, err := f.client.Project(f.project).Items().ListAll(ctx,
		WithLaunchID(launchID),
		WithStatus("FAILED"),
	)
	if err != nil {
		return nil, err
	}

	var failures []rca.FailureInfo
	for _, item := range items {
		fi := rca.FailureInfo{
			LaunchID: launchID,
			ItemID:   item.ID,
			ItemUUID: item.UUID,
			TestName: item.Name,
			Status:   item.Status,
		}
		if item.Issue != nil {
			fi.IssueType = item.Issue.IssueType
			fi.AutoAnalyzed = item.Issue.AutoAnalyzed
		}
		failures = append(failures, fi)
	}
	return failures, nil
}
