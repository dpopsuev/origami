package observability

import (
	"os"

	framework "github.com/dpopsuev/origami"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
)

// DefaultObservability returns a set of observers with zero configuration.
// OTel tracing uses the global tracer provider (configure OTEL_EXPORTER_OTLP_ENDPOINT
// for real export; noop otherwise). Prometheus uses the default registry.
func DefaultObservability() []framework.WalkObserver {
	tracer := otel.Tracer("origami")
	otelObs := NewOTelObserver(tracer)
	promObs := NewPrometheusCollector(prometheus.DefaultRegisterer.(*prometheus.Registry))
	return []framework.WalkObserver{otelObs, promObs}
}

// DefaultObservabilityWithRegistry returns observers using a custom Prometheus registry.
func DefaultObservabilityWithRegistry(reg *prometheus.Registry) []framework.WalkObserver {
	tracer := otel.Tracer("origami")
	otelObs := NewOTelObserver(tracer)
	promObs := NewPrometheusCollector(reg)
	return []framework.WalkObserver{otelObs, promObs}
}

// HasOTelEndpoint returns true if the OTLP endpoint environment variable is set.
func HasOTelEndpoint() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}
