package probes

import (
	"strings"

	"github.com/dpopsuev/origami/ouroboros"
)

// PersistenceInput presents a task where the obvious first approach fails.
// The agent must recognize the failure and try a different strategy.
const PersistenceInput = `=== Task: Parse the configuration file ===

You need to parse this configuration and extract all key-value pairs:

` + "```" + `
# Application config (v3 format)
database.host = "localhost"
database.port = 5432
database.name = "myapp"

# Feature flags use a different syntax
[features]
enable_cache: true
enable_metrics: true
max_retries: 3

# Secrets are base64-encoded inline
secrets.api_key = "base64:SGVsbG8gV29ybGQ="
secrets.jwt_secret = "base64:c2VjcmV0MTIz"

# Environment overrides (${VAR} syntax)
server.host = "${HOST:-0.0.0.0}"
server.port = "${PORT:-8080}"
` + "```" + `

CONSTRAINT: You must use Go's standard library only (no third-party config parsers).

Write a Go function: func ParseConfig(input string) (map[string]interface{}, error)

The function must handle ALL four formats:
1. key = "value" (properties style)
2. [section] + key: value (INI style)  
3. base64:XXX values (decode them)
4. ${VAR:-default} (resolve to default value)

Note: A simple line-by-line split on "=" will NOT work because of the mixed formats.
Start with whatever approach you think is best.`

// PersistenceSpec returns the ProbeSpec for the persistence probe.
func PersistenceSpec() ouroboros.ProbeSpec {
	return ouroboros.ProbeSpec{
		ID:          "persistence-v1",
		Name:        "Persistence Probe",
		Description: "Multi-format parser that defeats naive approaches. Measures retry behavior and approach variation.",
		Step:        ouroboros.StepPersistence,
		Dimensions: []ouroboros.Dimension{
			ouroboros.DimPersistence,
			ouroboros.DimConvergenceThreshold,
		},
		Input: PersistenceInput,
		ExpectedBehaviors: []string{
			"handles all 4 formats",
			"does not rely solely on splitting on '='",
			"handles base64 decoding",
			"handles environment variable defaults",
		},
	}
}

// PersistencePrompt returns the prompt text for the persistence probe.
func PersistencePrompt() string {
	return PersistenceInput
}

// ScorePersistence maps persistence output to behavioral dimension scores.
//
// Scoring signals:
//   - Multiple approaches/sections in code -> high persistence
//   - Handles all 4 formats -> high convergence
//   - Single naive approach -> low persistence, low convergence
//   - Error handling present -> higher convergence
func ScorePersistence(raw string) map[ouroboros.Dimension]float64 {
	lower := strings.ToLower(raw)

	handlesProperties := containsAny(lower, "properties", "key = ", "split", "strings.cut", "strings.splitn")
	handlesINI := containsAny(lower, "[section", "ini", "section", "currentSection", "current_section", "currentsection")
	handlesBase64 := containsAny(lower, "base64", "stdencoding", "decodestring", "decode")
	handlesEnvVar := containsAny(lower, "${", "env", "default", "os.getenv", "strings.trimprefix")

	formatCount := 0
	for _, handled := range []bool{handlesProperties, handlesINI, handlesBase64, handlesEnvVar} {
		if handled {
			formatCount++
		}
	}

	funcCount := strings.Count(lower, "func ")
	hasErrorHandling := containsAny(lower, "error", "err ", "err)", "fmt.errorf")

	persistence := float64(funcCount) * 0.15
	if funcCount >= 3 {
		persistence = 0.6
	}
	if formatCount >= 3 {
		persistence += 0.2
	}
	if hasErrorHandling {
		persistence += 0.1
	}

	convergence := float64(formatCount) * 0.2
	if hasErrorHandling {
		convergence += 0.15
	}
	if funcCount >= 2 {
		convergence += 0.05
	}

	return map[ouroboros.Dimension]float64{
		ouroboros.DimPersistence:          clamp(persistence),
		ouroboros.DimConvergenceThreshold: clamp(convergence),
	}
}
