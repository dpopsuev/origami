package rp

import (
	"context"
	"time"

	"github.com/dpopsuev/origami/schematics/rca"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
)

var _ rca.SourceAdapter = (*SourceAdapterRP)(nil)

// SourceAdapterRP implements rca.SourceAdapter for ReportPortal.
type SourceAdapterRP struct {
	client  *Client
	project string
}

// NewSourceAdapter creates a SourceAdapter connected to a ReportPortal instance.
func NewSourceAdapter(baseURL, apiKeyPath, project string) (rca.SourceAdapter, error) {
	key, err := ReadAPIKey(apiKeyPath)
	if err != nil {
		return nil, err
	}
	client, err := New(baseURL, key, WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	return &SourceAdapterRP{client: client, project: project}, nil
}

func (a *SourceAdapterRP) FetchEnvelope(launchID int) (*rcatype.Envelope, error) {
	f := NewFetcher(a.client, a.project)
	rpEnv, err := f.Fetch(launchID)
	if err != nil {
		return nil, err
	}
	return envelopeToRCAType(rpEnv), nil
}

func (a *SourceAdapterRP) EnvelopeFetcher() rcatype.EnvelopeFetcher {
	return &envelopeFetcherBridge{client: a.client, project: a.project}
}

func (a *SourceAdapterRP) CurrentUser() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if u, err := a.client.GetCurrentUser(ctx); err == nil && u.UserID != "" {
		return u.UserID
	}
	return ""
}

type envelopeFetcherBridge struct {
	client  *Client
	project string
}

func (b *envelopeFetcherBridge) Fetch(launchID int) (*rcatype.Envelope, error) {
	f := NewFetcher(b.client, b.project)
	rpEnv, err := f.Fetch(launchID)
	if err != nil {
		return nil, err
	}
	return envelopeToRCAType(rpEnv), nil
}

// DefectPusherRP implements rca.DefectPusher for ReportPortal.
type DefectPusherRP struct {
	pusher *Pusher
}

var _ rca.DefectPusher = (*DefectPusherRP)(nil)

func NewDefectPusher(baseURL, apiKeyPath, project, submittedBy string) (rca.DefectPusher, error) {
	key, err := ReadAPIKey(apiKeyPath)
	if err != nil {
		return nil, err
	}
	client, err := New(baseURL, key, WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	return &DefectPusherRP{pusher: NewPusher(client, project, submittedBy)}, nil
}

func (p *DefectPusherRP) Push(artifactPath, jiraTicketID, jiraLink string) (*rca.PushedRecord, error) {
	st := NewMemPushStore()
	if err := p.pusher.Push(artifactPath, st, jiraTicketID, jiraLink); err != nil {
		return nil, err
	}
	rec := st.LastPushed()
	if rec == nil {
		return nil, nil
	}
	return &rca.PushedRecord{LaunchID: rec.LaunchID, DefectType: rec.DefectType}, nil
}

// Conversion from rp types to rcatype — mirrors rpconv.EnvelopeFromRP but
// lives here to avoid an import cycle (rpconv imports connectors/rp).

func envelopeToRCAType(e *Envelope) *rcatype.Envelope {
	if e == nil {
		return nil
	}
	env := &rcatype.Envelope{
		RunID:      e.RunID,
		LaunchUUID: e.LaunchUUID,
		Name:       e.Name,
	}
	for _, f := range e.FailureList {
		item := rcatype.FailureItem{
			ID: f.ID, UUID: f.UUID, Name: f.Name, Type: f.Type,
			Status: f.Status, Path: f.Path, CodeRef: f.CodeRef,
			Description: f.Description, ParentID: f.ParentID,
			IssueType: f.IssueType, IssueComment: f.IssueComment,
			AutoAnalyzed: f.AutoAnalyzed,
		}
		for _, ei := range f.ExternalIssues {
			item.ExternalIssues = append(item.ExternalIssues, rcatype.ExternalIssue{
				TicketID: ei.TicketID, URL: ei.URL,
			})
		}
		env.FailureList = append(env.FailureList, item)
	}
	for _, a := range e.LaunchAttributes {
		env.LaunchAttributes = append(env.LaunchAttributes, rcatype.Attribute{
			Key: a.Key, Value: a.Value, System: a.System,
		})
	}
	return env
}
