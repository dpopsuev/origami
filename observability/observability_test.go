package observability

import (
	"context"
	"fmt"
	"testing"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestOTelObserver_SpanTree(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	obs := NewOTelObserver(tracer)

	obs.StartWalk("test-pipeline", attribute.String("element", "fire"))

	obs.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "recall", Walker: "w1"})
	obs.OnEvent(framework.WalkEvent{Type: framework.EventNodeExit, Node: "recall", Elapsed: 100 * time.Millisecond})
	obs.OnEvent(framework.WalkEvent{Type: framework.EventTransition, Edge: "e1", Node: "triage"})
	obs.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "triage", Walker: "w1"})
	obs.OnEvent(framework.WalkEvent{Type: framework.EventNodeExit, Node: "triage", Elapsed: 200 * time.Millisecond})
	obs.OnEvent(framework.WalkEvent{Type: framework.EventWalkComplete})

	spans := exporter.GetSpans()
	if len(spans) < 3 {
		t.Fatalf("expected at least 3 spans (walk + 2 nodes), got %d", len(spans))
	}

	var walkSpan, recallSpan, triageSpan *tracetest.SpanStub
	for i := range spans {
		switch spans[i].Name {
		case "pipeline.walk":
			walkSpan = &spans[i]
		case "node.visit":
			for _, a := range spans[i].Attributes {
				if a.Key == "node" {
					switch a.Value.AsString() {
					case "recall":
						recallSpan = &spans[i]
					case "triage":
						triageSpan = &spans[i]
					}
				}
			}
		}
	}

	if walkSpan == nil {
		t.Fatal("missing pipeline.walk root span")
	}
	if recallSpan == nil {
		t.Fatal("missing recall node span")
	}
	if triageSpan == nil {
		t.Fatal("missing triage node span")
	}

	// Node spans should be children of walk span
	if recallSpan.Parent.TraceID() != walkSpan.SpanContext.TraceID() {
		t.Error("recall span not child of walk span")
	}
	if triageSpan.Parent.TraceID() != walkSpan.SpanContext.TraceID() {
		t.Error("triage span not child of walk span")
	}

	// Walk span should have transition event
	foundTransition := false
	for _, ev := range walkSpan.Events {
		if ev.Name == "edge.transition" {
			foundTransition = true
		}
	}
	if !foundTransition {
		t.Error("walk span missing edge.transition event")
	}
}

func TestOTelObserver_WalkError(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	obs := NewOTelObserver(tracer)

	obs.StartWalk("error-pipeline")
	obs.OnEvent(framework.WalkEvent{
		Type:  framework.EventWalkError,
		Error: fmt.Errorf("node failed"),
	})

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	foundError := false
	for _, ev := range spans[0].Events {
		if ev.Name == "exception" {
			foundError = true
		}
	}
	if !foundError {
		t.Error("walk error span missing recorded error event")
	}
}

func TestPrometheusCollector_Metrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	col := NewPrometheusCollector(reg)

	col.StartWalk("my-pipeline")

	col.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "recall"})
	col.OnEvent(framework.WalkEvent{Type: framework.EventNodeExit, Node: "recall", Elapsed: 150 * time.Millisecond})
	col.OnEvent(framework.WalkEvent{Type: framework.EventTransition, Node: "recall", Edge: "e1",
		Metadata: map[string]any{"from": "recall", "to": "triage"}})
	col.OnEvent(framework.WalkEvent{Type: framework.EventNodeEnter, Node: "triage"})
	col.OnEvent(framework.WalkEvent{Type: framework.EventNodeExit, Node: "triage", Elapsed: 200 * time.Millisecond})
	col.OnEvent(framework.WalkEvent{Type: framework.EventWalkComplete})

	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	findMetric := func(name string) *dto.MetricFamily {
		for _, f := range families {
			if f.GetName() == name {
				return f
			}
		}
		return nil
	}

	// Node duration histogram should have 2 observations
	dur := findMetric("origami_walk_node_duration_seconds")
	if dur == nil {
		t.Fatal("missing origami_walk_node_duration_seconds")
	}
	totalCount := uint64(0)
	for _, m := range dur.GetMetric() {
		totalCount += m.GetHistogram().GetSampleCount()
	}
	if totalCount != 2 {
		t.Errorf("node duration sample count = %d, want 2", totalCount)
	}

	// Edge transitions
	edges := findMetric("origami_walk_edge_transitions_total")
	if edges == nil {
		t.Fatal("missing origami_walk_edge_transitions_total")
	}
	edgeTotal := 0.0
	for _, m := range edges.GetMetric() {
		edgeTotal += m.GetCounter().GetValue()
	}
	if edgeTotal != 1 {
		t.Errorf("edge transitions = %v, want 1", edgeTotal)
	}

	// Walk completed
	completed := findMetric("origami_walk_completed_total")
	if completed == nil {
		t.Fatal("missing origami_walk_completed_total")
	}
	completedTotal := 0.0
	for _, m := range completed.GetMetric() {
		completedTotal += m.GetCounter().GetValue()
	}
	if completedTotal != 1 {
		t.Errorf("walk completed = %v, want 1", completedTotal)
	}
}

func TestPrometheusCollector_ErrorStatus(t *testing.T) {
	reg := prometheus.NewRegistry()
	col := NewPrometheusCollector(reg)

	col.StartWalk("fail-pipeline")
	col.OnEvent(framework.WalkEvent{
		Type:  framework.EventWalkError,
		Error: fmt.Errorf("boom"),
	})

	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range families {
		if f.GetName() == "origami_walk_completed_total" {
			for _, m := range f.GetMetric() {
				for _, lp := range m.GetLabel() {
					if lp.GetName() == "status" && lp.GetValue() == "error" {
						if m.GetCounter().GetValue() == 1 {
							return
						}
					}
				}
			}
		}
	}
	t.Error("expected walk_completed_total with status=error")
}

func TestDefaultObservability_ReturnsTwoObservers(t *testing.T) {
	reg := prometheus.NewRegistry()
	obs := DefaultObservabilityWithRegistry(reg)
	if len(obs) != 2 {
		t.Fatalf("expected 2 observers, got %d", len(obs))
	}

	// First should be OTel, second should be Prometheus
	if _, ok := obs[0].(*OTelObserver); !ok {
		t.Error("first observer should be *OTelObserver")
	}
	if _, ok := obs[1].(*PrometheusCollector); !ok {
		t.Error("second observer should be *PrometheusCollector")
	}
}
