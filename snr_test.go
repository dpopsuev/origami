package framework

import (
	"context"
	"testing"
)

func TestEvidenceSNR(t *testing.T) {
	cases := []struct {
		in, out int
		want    float64
	}{
		{10, 5, 0.5},
		{10, 10, 1.0},
		{10, 0, 0.0},
		{0, 5, 0.0},
		{0, 0, 0.0},
		{4, 3, 0.75},
	}
	for _, tc := range cases {
		got := EvidenceSNR(tc.in, tc.out)
		if got != tc.want {
			t.Errorf("EvidenceSNR(%d, %d) = %f, want %f", tc.in, tc.out, got, tc.want)
		}
	}
}

type countableStubArtifact struct {
	typ        string
	confidence float64
	inputN     int
	outputN    int
}

func (a *countableStubArtifact) Type() string        { return a.typ }
func (a *countableStubArtifact) Confidence() float64  { return a.confidence }
func (a *countableStubArtifact) Raw() any             { return nil }
func (a *countableStubArtifact) InputCount() int      { return a.inputN }
func (a *countableStubArtifact) OutputCount() int     { return a.outputN }

var _ CountableArtifact = (*countableStubArtifact)(nil)

func TestWalk_SNRAutoEmitted(t *testing.T) {
	art := &countableStubArtifact{typ: "filtered", confidence: 0.9, inputN: 100, outputN: 30}
	nodeA := &stubNode{name: "filter", artifact: art}
	edges := []Edge{&stubEdge{id: "A-done", from: "filter", to: "_done"}}

	tc := &TraceCollector{}
	g, err := NewGraph("snr-test", []Node{nodeA}, edges, nil, WithObserver(tc))
	if err != nil {
		t.Fatal(err)
	}

	w := &stubWalker{
		identity: AgentIdentity{PersonaName: "Solo"},
		state:    NewWalkerState("s1"),
	}
	if err := g.Walk(context.Background(), w, "filter"); err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	exits := tc.EventsOfType(EventNodeExit)
	if len(exits) != 1 {
		t.Fatalf("expected 1 node_exit, got %d", len(exits))
	}

	snrVal, ok := exits[0].Metadata["snr"].(float64)
	if !ok {
		t.Fatal("EventNodeExit missing snr metadata")
	}
	want := 0.3
	if snrVal != want {
		t.Errorf("snr = %f, want %f", snrVal, want)
	}
}

func TestWalk_SNRNotEmittedForNonCountable(t *testing.T) {
	nodeA := &stubNode{name: "A", artifact: &stubArtifact{typ: "plain", confidence: 0.5}}
	edges := []Edge{&stubEdge{id: "A-done", from: "A", to: "_done"}}

	tc := &TraceCollector{}
	g, err := NewGraph("no-snr", []Node{nodeA}, edges, nil, WithObserver(tc))
	if err != nil {
		t.Fatal(err)
	}

	w := &stubWalker{
		identity: AgentIdentity{PersonaName: "Solo"},
		state:    NewWalkerState("s1"),
	}
	if err := g.Walk(context.Background(), w, "A"); err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	exits := tc.EventsOfType(EventNodeExit)
	if len(exits) != 1 {
		t.Fatalf("expected 1 node_exit, got %d", len(exits))
	}

	if _, ok := exits[0].Metadata["snr"]; ok {
		t.Error("non-CountableArtifact should not emit snr metadata")
	}
}
