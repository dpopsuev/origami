package lint

// AllRules returns all built-in lint rules ordered by category:
// structural (S), semantic (G), best-practice (B).
func AllRules() []Rule {
	return append(append(structuralRules(), semanticRules()...), bestPracticeRules()...)
}

func structuralRules() []Rule {
	return []Rule{
		&MissingNodeElement{},
		&InvalidElement{},
		&InvalidMergeStrategy{},
		&MissingEdgeName{},
		&DuplicateEdgeCondition{},
		&EmptyPrompt{},
		&InvalidCacheTTL{},
		&MissingCircuitDescription{},
		&UnnamedNode{},
		&InvalidWalkerElement{},
		&InvalidWalkerPersona{},
		&SchemaInUnstructuredZone{},
		&MissingZoneDomain{},
		&InvalidZoneDomain{},
	}
}

func semanticRules() []Rule {
	return []Rule{
		&OrphanNode{},
		&UnreachableDone{},
		&DeadEdge{},
		&ShortcutBypassesRequired{},
		&ZoneElementMismatch{},
		&ExpressionCompileError{},
		&FanInWithoutMerge{},
	}
}

func bestPracticeRules() []Rule {
	return []Rule{
		&PreferWhenOverCondition{},
		&NameYourEdges{},
		&TerminalEdgeToDone{},
		&ZoneStickinessWithoutProvider{},
		&LargeCircuitNoZones{},
		&ElementAffinityChain{},
		&StochasticTransformer{},
	}
}
