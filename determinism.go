package framework

// IsCircuitDeterministic returns true if every node in the circuit that
// references a transformer resolves to a deterministic transformer.
// Nodes without a transformer field (e.g., custom nodes)
// are skipped. Returns false if any transformer is unresolvable or stochastic.
func IsCircuitDeterministic(def *CircuitDef, reg TransformerRegistry) bool {
	if reg == nil {
		return false
	}
	for _, nd := range def.Nodes {
		if nd.Transformer == "" {
			continue
		}
		t, err := reg.Get(nd.Transformer)
		if err != nil {
			return false
		}
		if !IsDeterministic(t) {
			return false
		}
	}
	return true
}
