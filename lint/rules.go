package lint

// AllRules returns all built-in lint rules ordered by category:
// structural (S), semantic (G), best-practice (B), prompt (P).
func AllRules() []Rule {
	all := append(structuralRules(), semanticRules()...)
	all = append(all, bestPracticeRules()...)
	all = append(all, promptRules()...)
	return all
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
		&StochasticSummary{},
	}
}

func promptRules() []Rule {
	return []Rule{
		&TemplateParamValidity{},
	}
}
