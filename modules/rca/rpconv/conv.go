// Package rpconv converts between RCA domain types (rcatype) and
// ReportPortal types (components/rp). All RP-specific logic stays at the
// CLI/MCP boundary; the RCA core works exclusively with rcatype.
package rpconv

import (
	"github.com/dpopsuev/origami/components/rp"
	"github.com/dpopsuev/origami/modules/rca/rcatype"
	"github.com/dpopsuev/origami/modules/rca/store"
)

// EnvelopeFromRP converts an rp.Envelope to an rcatype.Envelope.
func EnvelopeFromRP(e *rp.Envelope) *rcatype.Envelope {
	if e == nil {
		return nil
	}
	return &rcatype.Envelope{
		RunID:            e.RunID,
		LaunchUUID:       e.LaunchUUID,
		Name:             e.Name,
		FailureList:      failureItemsFromRP(e.FailureList),
		LaunchAttributes: attributesFromRP(e.LaunchAttributes),
	}
}

// EnvelopeToRP converts an rcatype.Envelope to an rp.Envelope.
func EnvelopeToRP(e *rcatype.Envelope) *rp.Envelope {
	if e == nil {
		return nil
	}
	return &rp.Envelope{
		RunID:            e.RunID,
		LaunchUUID:       e.LaunchUUID,
		Name:             e.Name,
		FailureList:      failureItemsToRP(e.FailureList),
		LaunchAttributes: attributesToRP(e.LaunchAttributes),
	}
}

func failureItemsFromRP(items []rp.FailureItem) []rcatype.FailureItem {
	if items == nil {
		return nil
	}
	out := make([]rcatype.FailureItem, len(items))
	for i, f := range items {
		out[i] = rcatype.FailureItem{
			ID:             f.ID,
			UUID:           f.UUID,
			Name:           f.Name,
			Type:           f.Type,
			Status:         f.Status,
			Path:           f.Path,
			CodeRef:        f.CodeRef,
			Description:    f.Description,
			ParentID:       f.ParentID,
			IssueType:      f.IssueType,
			IssueComment:   f.IssueComment,
			AutoAnalyzed:   f.AutoAnalyzed,
			ExternalIssues: externalIssuesFromRP(f.ExternalIssues),
		}
	}
	return out
}

func failureItemsToRP(items []rcatype.FailureItem) []rp.FailureItem {
	if items == nil {
		return nil
	}
	out := make([]rp.FailureItem, len(items))
	for i, f := range items {
		out[i] = rp.FailureItem{
			ID:             f.ID,
			UUID:           f.UUID,
			Name:           f.Name,
			Type:           f.Type,
			Status:         f.Status,
			Path:           f.Path,
			CodeRef:        f.CodeRef,
			Description:    f.Description,
			ParentID:       f.ParentID,
			IssueType:      f.IssueType,
			IssueComment:   f.IssueComment,
			AutoAnalyzed:   f.AutoAnalyzed,
			ExternalIssues: externalIssuesToRP(f.ExternalIssues),
		}
	}
	return out
}

func attributesFromRP(attrs []rp.Attribute) []rcatype.Attribute {
	if attrs == nil {
		return nil
	}
	out := make([]rcatype.Attribute, len(attrs))
	for i, a := range attrs {
		out[i] = rcatype.Attribute{Key: a.Key, Value: a.Value, System: a.System}
	}
	return out
}

func attributesToRP(attrs []rcatype.Attribute) []rp.Attribute {
	if attrs == nil {
		return nil
	}
	out := make([]rp.Attribute, len(attrs))
	for i, a := range attrs {
		out[i] = rp.Attribute{Key: a.Key, Value: a.Value, System: a.System}
	}
	return out
}

func externalIssuesFromRP(issues []rp.ExternalIssue) []rcatype.ExternalIssue {
	if issues == nil {
		return nil
	}
	out := make([]rcatype.ExternalIssue, len(issues))
	for i, e := range issues {
		out[i] = rcatype.ExternalIssue{TicketID: e.TicketID, URL: e.URL}
	}
	return out
}

func externalIssuesToRP(issues []rcatype.ExternalIssue) []rp.ExternalIssue {
	if issues == nil {
		return nil
	}
	out := make([]rp.ExternalIssue, len(issues))
	for i, e := range issues {
		out[i] = rp.ExternalIssue{TicketID: e.TicketID, URL: e.URL}
	}
	return out
}

// RPFetcherAdapter wraps an rp.EnvelopeFetcher to implement rcatype.EnvelopeFetcher.
type RPFetcherAdapter struct {
	Inner rp.EnvelopeFetcher
}

// Fetch implements rcatype.EnvelopeFetcher.
func (a *RPFetcherAdapter) Fetch(launchID int) (*rcatype.Envelope, error) {
	rpEnv, err := a.Inner.Fetch(launchID)
	if err != nil {
		return nil, err
	}
	return EnvelopeFromRP(rpEnv), nil
}

// EnvelopeStoreAdapter bridges an RCA store (rcatype.Envelope) with
// rp.EnvelopeStore (rp.Envelope), converting at the boundary.
// Used by rp.FetchAndSave to persist RP-fetched envelopes into the RCA store.
type EnvelopeStoreAdapter struct {
	Store store.Store
}

// Save implements rp.EnvelopeStore by converting rp.Envelope to rcatype.Envelope.
func (a *EnvelopeStoreAdapter) Save(launchID int, envelope *rp.Envelope) error {
	return a.Store.SaveEnvelope(launchID, EnvelopeFromRP(envelope))
}

// Get implements rp.EnvelopeStore by converting rcatype.Envelope to rp.Envelope.
func (a *EnvelopeStoreAdapter) Get(launchID int) (*rp.Envelope, error) {
	env, err := a.Store.GetEnvelope(launchID)
	if err != nil {
		return nil, err
	}
	return EnvelopeToRP(env), nil
}
