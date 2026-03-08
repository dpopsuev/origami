package framework
// Category: Processing & Support

// isCircuitDeterministic returns true if every node in the circuit that
// references a transformer resolves to a deterministic transformer.
// Nodes without a transformer are skipped.
// Returns false if any transformer is unresolvable or stochastic.
func isCircuitDeterministic(def *CircuitDef, reg TransformerRegistry) bool {
	if reg == nil {
		return false
	}
	for _, nd := range def.Nodes {
		ht := nd.EffectiveHandlerType(def.HandlerType)
		name := nd.EffectiveHandler()
		if ht != HandlerTypeTransformer || name == "" {
			continue
		}
		t, err := reg.Get(name)
		if err != nil {
			return false
		}
		if !IsDeterministic(t) {
			return false
		}
	}
	return true
}
