// Package rcatype defines RCA domain types decoupled from any data source.
// Both modules/rca and modules/rca/store import this package to avoid
// circular dependencies. Conversion to/from source-specific types (e.g.
// components/rp) happens in modules/rca/rpconv.
package rcatype

// Envelope is the execution envelope (launch + failure list).
type Envelope struct {
	RunID       string `json:"run_id"`
	LaunchUUID  string `json:"launch_uuid"`
	Name        string `json:"name"`
	FailureList []FailureItem `json:"failure_list"`

	LaunchAttributes []Attribute `json:"launch_attributes,omitempty"`
}

// Attribute is a key-value pair from launch or test item attributes.
type Attribute struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	System bool   `json:"system,omitempty"`
}

// FailureItem is one failed test step in the envelope.
type FailureItem struct {
	ID     int    `json:"id"`
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Path   string `json:"path"`

	CodeRef      string `json:"code_ref,omitempty"`
	Description  string `json:"description,omitempty"`
	ParentID     int    `json:"parent_id,omitempty"`
	IssueType    string `json:"issue_type,omitempty"`
	IssueComment string `json:"issue_comment,omitempty"`
	AutoAnalyzed bool   `json:"auto_analyzed,omitempty"`

	ExternalIssues []ExternalIssue `json:"external_issues,omitempty"`
}

// ExternalIssue links a test failure to an external bug tracker ticket.
type ExternalIssue struct {
	TicketID string `json:"ticket_id"`
	URL      string `json:"url,omitempty"`
}

// EnvelopeFetcher retrieves an envelope by launch ID.
type EnvelopeFetcher interface {
	Fetch(launchID int) (*Envelope, error)
}
