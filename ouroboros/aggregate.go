package ouroboros

import (
	"time"

	framework "github.com/dpopsuev/origami"
)

const SeedBatteryVersion = "ouroboros-seed-v1"

// PoleResultToProbeResult converts a judge-produced PoleResult into a
// ProbeResult suitable for dimension aggregation. This bridges the seed
// pipeline output into the existing ModelProfile aggregation path.
func PoleResultToProbeResult(seedName string, pr *PoleResult, elapsed time.Duration) ProbeResult {
	return ProbeResult{
		ProbeID:         seedName,
		RawOutput:       pr.Reasoning,
		DimensionScores: pr.DimensionScores,
		Elapsed:         elapsed,
	}
}

// ProfileFromPoleResults aggregates multiple seed pipeline PoleResults into
// a ModelProfile, using the same dimension averaging as the v1 runner.
// This replaces RunOuroboros for seed-pipeline workflows.
func ProfileFromPoleResults(
	model framework.ModelIdentity,
	results []PoleResult,
	seedNames []string,
) ModelProfile {
	profile := ModelProfile{
		Model:          model,
		BatteryVersion: SeedBatteryVersion,
		Timestamp:      time.Now(),
		Dimensions:     make(map[Dimension]float64),
		ElementScores:  make(map[framework.Element]float64),
	}

	for i, pr := range results {
		name := ""
		if i < len(seedNames) {
			name = seedNames[i]
		}
		profile.RawResults = append(profile.RawResults,
			PoleResultToProbeResult(name, &pr, 0))
	}

	aggregateDimensions(&profile)

	profile.ElementMatch = ElementMatch(profile)
	profile.ElementScores = ElementScores(profile)
	profile.SuggestedPersonas = SuggestPersona(profile)

	return profile
}
