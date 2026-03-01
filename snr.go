package framework

// CountableArtifact is an optional extension of Artifact for nodes that
// process discrete items. When an artifact implements this interface, the
// walk loop auto-computes EvidenceSNR and emits it as "snr" metadata on
// EventNodeExit. Artifacts where item counts don't apply (classifications,
// verdicts, etc.) should not implement this.
type CountableArtifact interface {
	Artifact
	InputCount() int
	OutputCount() int
}

// EvidenceSNR computes the signal-to-noise ratio of a processing step.
// outputItems / inputItems gives the signal preservation ratio.
// Returns 0 if inputItems is 0 (no signal to measure).
//
// Auto-wired: when an Artifact implements CountableArtifact, the walk loop
// calls this and emits the result as metadata on EventNodeExit. The
// PrometheusCollector picks it up automatically. Consumers can also call
// this directly for manual computation.
func EvidenceSNR(inputItems, outputItems int) float64 {
	if inputItems <= 0 {
		return 0
	}
	return float64(outputItems) / float64(inputItems)
}
