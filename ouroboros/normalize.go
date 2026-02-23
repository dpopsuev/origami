package ouroboros

import (
	"sort"
)

// NormalizeProfile recomputes a profile's dimension scores as percentiles
// relative to all stored profiles. A percentile of 0.5 means median.
// The profile is modified in place. allProfiles must include the target.
func NormalizeProfile(profile *ModelProfile, allProfiles []ModelProfile) {
	if len(allProfiles) <= 1 {
		return
	}

	for _, dim := range AllDimensions() {
		raw, ok := profile.Dimensions[dim]
		if !ok {
			continue
		}

		var values []float64
		for _, p := range allProfiles {
			if v, exists := p.Dimensions[dim]; exists {
				values = append(values, v)
			}
		}

		if len(values) <= 1 {
			continue
		}

		profile.Dimensions[dim] = percentile(raw, values)
	}
}

// percentile returns the fraction of values that are <= the given value.
// With n values, the percentile is (count of values <= v) / n.
func percentile(v float64, values []float64) float64 {
	sort.Float64s(values)
	count := 0
	for _, val := range values {
		if val <= v {
			count++
		}
	}
	return float64(count) / float64(len(values))
}

// IsStale returns true if a profile should be re-calibrated. A profile
// is stale if its battery version differs from the current version.
func IsStale(profile ModelProfile, currentBattery string) bool {
	return profile.BatteryVersion != currentBattery
}

// DimensionDelta records how a dimension changed between two profiles.
type DimensionDelta struct {
	Dimension Dimension `json:"dimension"`
	Before    float64   `json:"before"`
	After     float64   `json:"after"`
	Delta     float64   `json:"delta"`
}

// CompareVersions compares the first and last profiles in a chronological
// slice and returns the per-dimension deltas. Profiles must be sorted
// oldest-first (as returned by ProfileStore.History).
func CompareVersions(profiles []ModelProfile) []DimensionDelta {
	if len(profiles) < 2 {
		return nil
	}

	first := profiles[0]
	last := profiles[len(profiles)-1]

	var deltas []DimensionDelta
	for _, dim := range AllDimensions() {
		before, bOK := first.Dimensions[dim]
		after, aOK := last.Dimensions[dim]
		if !bOK || !aOK {
			continue
		}
		deltas = append(deltas, DimensionDelta{
			Dimension: dim,
			Before:    before,
			After:     after,
			Delta:     after - before,
		})
	}

	return deltas
}
