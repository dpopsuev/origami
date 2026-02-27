package observability

import (
	"sync"
	"time"

	framework "github.com/dpopsuev/origami"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector translates WalkEvents into Prometheus metrics.
// It also exposes RecordTokens and RecordDispatch for bridging with
// the dispatch layer without creating import cycles.
type PrometheusCollector struct {
	NodeDuration     *prometheus.HistogramVec
	EdgeTransitions  *prometheus.CounterVec
	WalkActive       *prometheus.GaugeVec
	WalkCompleted    *prometheus.CounterVec
	LoopsTotal       *prometheus.CounterVec

	TokensTotal      *prometheus.CounterVec
	TokensCostUSD    *prometheus.CounterVec

	DispatchDuration *prometheus.HistogramVec
	DispatchErrors   *prometheus.CounterVec

	Registry *prometheus.Registry

	mu       sync.Mutex
	pipeline string
}

// NewPrometheusCollector creates a collector and registers metrics on the given registry.
// Pass nil to use a new default registry.
func NewPrometheusCollector(reg *prometheus.Registry) *PrometheusCollector {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	c := &PrometheusCollector{
		Registry: reg,
		NodeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "origami_walk_node_duration_seconds",
			Help:    "Duration of node processing in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"pipeline", "node"}),
		EdgeTransitions: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_walk_edge_transitions_total",
			Help: "Total edge transitions.",
		}, []string{"pipeline", "from", "to"}),
		WalkActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "origami_walk_active",
			Help: "Number of active walks.",
		}, []string{"pipeline"}),
		WalkCompleted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_walk_completed_total",
			Help: "Total completed walks.",
		}, []string{"pipeline", "status"}),
		LoopsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_walk_loops_total",
			Help: "Total loop iterations.",
		}, []string{"pipeline", "node"}),

		TokensTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_tokens_total",
			Help: "Total LLM tokens consumed.",
		}, []string{"pipeline", "step", "direction"}),
		TokensCostUSD: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_tokens_cost_usd",
			Help: "Estimated LLM token cost in USD.",
		}, []string{"pipeline", "step"}),

		DispatchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "origami_dispatch_duration_seconds",
			Help:    "Duration of dispatch calls in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}, []string{"provider", "step"}),
		DispatchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "origami_dispatch_errors_total",
			Help: "Total dispatch errors by provider and step.",
		}, []string{"provider", "step"}),
	}

	reg.MustRegister(
		c.NodeDuration, c.EdgeTransitions, c.WalkActive, c.WalkCompleted, c.LoopsTotal,
		c.TokensTotal, c.TokensCostUSD,
		c.DispatchDuration, c.DispatchErrors,
	)
	return c
}

// SetPipeline configures the pipeline label for subsequent events.
func (c *PrometheusCollector) SetPipeline(name string) {
	c.mu.Lock()
	c.pipeline = name
	c.mu.Unlock()
}

func (c *PrometheusCollector) OnEvent(e framework.WalkEvent) {
	c.mu.Lock()
	pipeline := c.pipeline
	c.mu.Unlock()

	switch e.Type {
	case framework.EventNodeExit:
		c.NodeDuration.WithLabelValues(pipeline, e.Node).Observe(e.Elapsed.Seconds())
	case framework.EventTransition:
		from := ""
		to := ""
		if e.Metadata != nil {
			if f, ok := e.Metadata["from"].(string); ok {
				from = f
			}
			if t, ok := e.Metadata["to"].(string); ok {
				to = t
			}
		}
		if from == "" {
			from = e.Node
		}
		c.EdgeTransitions.WithLabelValues(pipeline, from, to).Inc()
	case framework.EventWalkComplete:
		c.WalkActive.WithLabelValues(pipeline).Dec()
		c.WalkCompleted.WithLabelValues(pipeline, "success").Inc()
	case framework.EventWalkError:
		c.WalkActive.WithLabelValues(pipeline).Dec()
		c.WalkCompleted.WithLabelValues(pipeline, "error").Inc()
	case framework.EventNodeEnter:
		if pipeline != "" {
			c.WalkActive.WithLabelValues(pipeline).Add(0) // ensure label exists
		}
	}
}

// StartWalk increments the active walk gauge.
func (c *PrometheusCollector) StartWalk(pipeline string) {
	c.SetPipeline(pipeline)
	c.WalkActive.WithLabelValues(pipeline).Inc()
}

// RecordTokens increments token counters for a dispatch step.
// direction should be "prompt" or "artifact".
func (c *PrometheusCollector) RecordTokens(step string, promptTokens, artifactTokens int, costUSD float64) {
	c.mu.Lock()
	pipeline := c.pipeline
	c.mu.Unlock()

	c.TokensTotal.WithLabelValues(pipeline, step, "prompt").Add(float64(promptTokens))
	c.TokensTotal.WithLabelValues(pipeline, step, "artifact").Add(float64(artifactTokens))
	c.TokensCostUSD.WithLabelValues(pipeline, step).Add(costUSD)
}

// RecordDispatch records a dispatch duration and optional error.
func (c *PrometheusCollector) RecordDispatch(provider, step string, duration time.Duration, err error) {
	c.DispatchDuration.WithLabelValues(provider, step).Observe(duration.Seconds())
	if err != nil {
		c.DispatchErrors.WithLabelValues(provider, step).Inc()
	}
}
