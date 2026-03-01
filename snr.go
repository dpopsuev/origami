package framework

// EvidenceSNR computes the signal-to-noise ratio of a processing step.
// outputItems / inputItems gives the signal preservation ratio.
// Returns 0 if inputItems is 0 (no signal to measure).
//
// Consumer-wired: the walk loop does not call this automatically because the
// Artifact interface has no InputCount/OutputCount methods. Consumers should
// call EvidenceSNR in their node implementations and report via Prometheus.
func EvidenceSNR(inputItems, outputItems int) float64 {
	if inputItems <= 0 {
		return 0
	}
	return float64(outputItems) / float64(inputItems)
}
