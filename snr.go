package framework

// EvidenceSNR computes the signal-to-noise ratio of a processing step.
// outputItems / inputItems gives the signal preservation ratio.
// Returns 0 if inputItems is 0 (no signal to measure).
func EvidenceSNR(inputItems, outputItems int) float64 {
	if inputItems <= 0 {
		return 0
	}
	return float64(outputItems) / float64(inputItems)
}
