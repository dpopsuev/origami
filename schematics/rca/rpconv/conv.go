// Package rpconv converts between RCA domain types (rcatype) and
// ReportPortal types (connectors/rp). All RP-specific logic stays at the
// CLI/MCP boundary; the RCA core works exclusively with rcatype.
package rpconv

import (
	"strconv"

	"github.com/dpopsuev/origami/connectors/rp"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/store"
)

// EnvelopeFromRP converts an rp.Envelope to an rcatype.Envelope.
// RP-specific values are stored in Tags with "rp." prefix.
func EnvelopeFromRP(e *rp.Envelope) *rcatype.Envelope {
	if e == nil {
		return nil
	}
	tags := map[string]string{}
	if e.LaunchUUID != "" {
		tags["rp.launch_uuid"] = e.LaunchUUID
	}
	env := &rcatype.Envelope{
		RunID:            e.RunID,
		Name:             e.Name,
		FailureList:      failureItemsFromRP(e.FailureList),
		LaunchAttributes: attributesFromRP(e.LaunchAttributes),
	}
	if len(tags) > 0 {
		env.Tags = tags
	}
	return env
}

// EnvelopeToRP converts an rcatype.Envelope to an rp.Envelope.
// RP-specific values are read from Tags with "rp." prefix.
func EnvelopeToRP(e *rcatype.Envelope) *rp.Envelope {
	if e == nil {
		return nil
	}
	launchUUID := ""
	if e.Tags != nil {
		launchUUID = e.Tags["rp.launch_uuid"]
	}
	return &rp.Envelope{
		RunID:            e.RunID,
		LaunchUUID:       launchUUID,
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
		tags := map[string]string{}
		if f.UUID != "" {
			tags["rp.uuid"] = f.UUID
		}
		if f.Type != "" {
			tags["rp.type"] = f.Type
		}
		if f.Path != "" {
			tags["rp.path"] = f.Path
		}
		if f.CodeRef != "" {
			tags["rp.code_ref"] = f.CodeRef
		}
		if f.ParentID != 0 {
			tags["rp.parent_id"] = strconv.Itoa(f.ParentID)
		}
		if f.IssueType != "" {
			tags["rp.issue_type"] = f.IssueType
		}
		if f.AutoAnalyzed {
			tags["rp.auto_analyzed"] = "true"
		}
		fi := rcatype.FailureItem{
			ID:             strconv.Itoa(f.ID),
			Name:           f.Name,
			Status:         f.Status,
			Description:    f.Description,
			ErrorMessage:   f.Description,
			LogSnippet:     f.IssueComment,
			ExternalIssues: externalIssuesFromRP(f.ExternalIssues),
		}
		if len(tags) > 0 {
			fi.Tags = tags
		}
		out[i] = fi
	}
	return out
}

func failureItemsToRP(items []rcatype.FailureItem) []rp.FailureItem {
	if items == nil {
		return nil
	}
	out := make([]rp.FailureItem, len(items))
	for i, f := range items {
		id, _ := strconv.Atoi(f.ID)
		parentID := 0
		issueType := ""
		autoAnalyzed := false
		uuid := ""
		typ := ""
		path := ""
		codeRef := ""
		issueComment := f.LogSnippet
		if f.Tags != nil {
			uuid = f.Tags["rp.uuid"]
			typ = f.Tags["rp.type"]
			path = f.Tags["rp.path"]
			codeRef = f.Tags["rp.code_ref"]
			parentID, _ = strconv.Atoi(f.Tags["rp.parent_id"])
			issueType = f.Tags["rp.issue_type"]
			autoAnalyzed = f.Tags["rp.auto_analyzed"] == "true"
		}
		out[i] = rp.FailureItem{
			ID:             id,
			UUID:           uuid,
			Name:           f.Name,
			Type:           typ,
			Status:         f.Status,
			Path:           path,
			CodeRef:        codeRef,
			Description:    f.Description,
			ParentID:       parentID,
			IssueType:      issueType,
			IssueComment:   issueComment,
			AutoAnalyzed:   autoAnalyzed,
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
func (a *RPFetcherAdapter) Fetch(runID string) (*rcatype.Envelope, error) {
	launchID, _ := strconv.Atoi(runID)
	rpEnv, err := a.Inner.Fetch(launchID)
	if err != nil {
		return nil, err
	}
	return EnvelopeFromRP(rpEnv), nil
}

// EnvelopeStoreAdapter bridges an RCA store (rcatype.Envelope) with
// rp.EnvelopeStore (rp.Envelope), converting at the boundary.
type EnvelopeStoreAdapter struct {
	Store store.Store
}

// Save implements rp.EnvelopeStore by converting rp.Envelope to rcatype.Envelope.
func (a *EnvelopeStoreAdapter) Save(launchID int, envelope *rp.Envelope) error {
	return a.Store.SaveEnvelope(strconv.Itoa(launchID), EnvelopeFromRP(envelope))
}

// Get implements rp.EnvelopeStore by converting rcatype.Envelope to rp.Envelope.
func (a *EnvelopeStoreAdapter) Get(launchID int) (*rp.Envelope, error) {
	env, err := a.Store.GetEnvelope(strconv.Itoa(launchID))
	if err != nil {
		return nil, err
	}
	return EnvelopeToRP(env), nil
}
