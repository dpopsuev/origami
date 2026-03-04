package rpconv

import (
	"path/filepath"
	"testing"

	"github.com/dpopsuev/origami/connectors/rp"
	"github.com/dpopsuev/origami/schematics/rca/rcatype"
	"github.com/dpopsuev/origami/schematics/rca/store"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	orig := &rp.Envelope{
		RunID:      "33195",
		LaunchUUID: "uuid-abc",
		Name:       "test-launch",
		FailureList: []rp.FailureItem{
			{
				ID: 1, UUID: "f1", Name: "fail1", Type: "STEP", Status: "FAILED", Path: "/suite/test1",
				CodeRef: "ref1", Description: "desc1", ParentID: 10,
				IssueType: "ti001", IssueComment: "comment1", AutoAnalyzed: true,
				ExternalIssues: []rp.ExternalIssue{{TicketID: "JIRA-1", URL: "https://jira/1"}},
			},
			{ID: 2, Name: "fail2", Status: "FAILED"},
		},
		LaunchAttributes: []rp.Attribute{
			{Key: "ocp_version", Value: "4.17", System: false},
			{Key: "cluster", Value: "cnfde22", System: true},
		},
	}

	rcaEnv := EnvelopeFromRP(orig)
	if rcaEnv == nil {
		t.Fatal("EnvelopeFromRP returned nil")
	}
	if rcaEnv.RunID != orig.RunID {
		t.Errorf("RunID: got %q want %q", rcaEnv.RunID, orig.RunID)
	}
	if len(rcaEnv.FailureList) != 2 {
		t.Fatalf("FailureList: got %d want 2", len(rcaEnv.FailureList))
	}
	if rcaEnv.FailureList[0].ExternalIssues[0].TicketID != "JIRA-1" {
		t.Error("ExternalIssues not preserved")
	}

	back := EnvelopeToRP(rcaEnv)
	if back == nil {
		t.Fatal("EnvelopeToRP returned nil")
	}
	if back.RunID != orig.RunID {
		t.Errorf("round-trip RunID: got %q want %q", back.RunID, orig.RunID)
	}
	if len(back.FailureList) != len(orig.FailureList) {
		t.Errorf("round-trip FailureList len: got %d want %d", len(back.FailureList), len(orig.FailureList))
	}
	if back.FailureList[0].AutoAnalyzed != orig.FailureList[0].AutoAnalyzed {
		t.Error("round-trip AutoAnalyzed mismatch")
	}
	if len(back.LaunchAttributes) != len(orig.LaunchAttributes) {
		t.Errorf("round-trip LaunchAttributes: got %d want %d", len(back.LaunchAttributes), len(orig.LaunchAttributes))
	}
}

func TestEnvelopeFromRP_Nil(t *testing.T) {
	if got := EnvelopeFromRP(nil); got != nil {
		t.Errorf("EnvelopeFromRP(nil): got %v want nil", got)
	}
	if got := EnvelopeToRP(nil); got != nil {
		t.Errorf("EnvelopeToRP(nil): got %v want nil", got)
	}
}

func TestRPFetcherAdapter(t *testing.T) {
	env := &rp.Envelope{RunID: "42", Name: "stub-launch"}
	stub := &rp.StubFetcher{Env: env}
	adapter := &RPFetcherAdapter{Inner: stub}

	got, err := adapter.Fetch(42)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.RunID != "42" {
		t.Errorf("RunID: got %q want %q", got.RunID, "42")
	}

	var _ rcatype.EnvelopeFetcher = adapter
}

func TestEnvelopeStoreAdapter_WithSqlStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	adapter := &EnvelopeStoreAdapter{Store: s}
	rpEnv := &rp.Envelope{RunID: "99", Name: "adapter-test", FailureList: nil}

	if err := adapter.Save(99, rpEnv); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := adapter.Get(99)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.RunID != "99" {
		t.Errorf("Get: got %+v", got)
	}

	rcaEnv, err := s.GetEnvelope(99)
	if err != nil {
		t.Fatalf("store.GetEnvelope: %v", err)
	}
	if rcaEnv == nil || rcaEnv.RunID != "99" {
		t.Errorf("store.GetEnvelope: got %+v", rcaEnv)
	}
}
