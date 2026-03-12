package calibrate

import (
	framework "github.com/dpopsuev/origami"
)

// Resolution defines a calibration scope level.
type Resolution string

const (
	ResolutionUnit       Resolution = "unit"       // single circuit in isolation
	ResolutionPairwise   Resolution = "pairwise"   // two circuits composed via ports
	ResolutionIntegrated Resolution = "integrated"  // full end-to-end composition
)

// MultiResolutionConfig defines a calibration plan across multiple resolutions.
type MultiResolutionConfig struct {
	Circuits []CircuitEntry   `yaml:"circuits"`
	Plans    []ResolutionPlan `yaml:"plans"`
}

// CircuitEntry names a circuit that participates in multi-resolution calibration.
type CircuitEntry struct {
	Name      string `yaml:"name"`
	Circuit   string `yaml:"circuit"`   // circuit name or import reference
	Scorecard string `yaml:"scorecard,omitempty"`
}

// ResolutionPlan describes one calibration resolution level.
type ResolutionPlan struct {
	Name       string     `yaml:"name"`
	Resolution Resolution `yaml:"resolution"`
	Circuits   []string   `yaml:"circuits"` // names from CircuitEntry
	Stubs      []StubDef  `yaml:"stubs,omitempty"`
}

// StubDef declares a port stub for isolated calibration.
// When calibrating a circuit in isolation, its port dependencies are
// replaced with canned data from the stub.
type StubDef struct {
	Port    string `yaml:"port"`    // "circuit.direction:port_name"
	Fixture string `yaml:"fixture"` // path to canned data file
}

// BuildResolutionPlans generates calibration plans for a set of circuits.
// Unit plans are generated for each circuit. Pairwise plans for circuits
// that share ports. Integrated plans for the full composition.
func BuildResolutionPlans(circuits []CircuitEntry) []ResolutionPlan {
	var plans []ResolutionPlan

	// Unit: each circuit independently
	for _, c := range circuits {
		plans = append(plans, ResolutionPlan{
			Name:       c.Name + "-unit",
			Resolution: ResolutionUnit,
			Circuits:   []string{c.Name},
		})
	}

	// Pairwise: each pair
	for i := 0; i < len(circuits); i++ {
		for j := i + 1; j < len(circuits); j++ {
			plans = append(plans, ResolutionPlan{
				Name:       circuits[i].Name + "-" + circuits[j].Name,
				Resolution: ResolutionPairwise,
				Circuits:   []string{circuits[i].Name, circuits[j].Name},
			})
		}
	}

	// Integrated: all circuits
	if len(circuits) > 1 {
		names := make([]string, len(circuits))
		for i, c := range circuits {
			names[i] = c.Name
		}
		plans = append(plans, ResolutionPlan{
			Name:       "integrated",
			Resolution: ResolutionIntegrated,
			Circuits:   names,
		})
	}

	return plans
}

// WrapForResolution decorates a circuit for a specific resolution level.
// For unit resolution, ports are stubbed with fixture data.
// For pairwise/integrated, circuits are composed via their port connections.
func WrapForResolution(base *framework.CircuitDef, plan ResolutionPlan, config DecoratorConfig) *framework.CircuitDef {
	wrapped := Wrap(base, config)

	if wrapped.Vars == nil {
		wrapped.Vars = make(map[string]any)
	}
	wrapped.Vars["_calibration_resolution"] = string(plan.Resolution)
	wrapped.Vars["_calibration_plan"] = plan.Name

	return wrapped
}
