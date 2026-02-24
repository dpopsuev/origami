package calibrate

import (
	"encoding/json"

	framework "github.com/dpopsuev/origami"
)

// ModelAdapter is the interface for sending prompts and receiving responses
// during calibration. The step parameter is a plain string so consumers
// can use their own step type (e.g. PipelineStep) without coupling the
// framework to a domain-specific enum.
type ModelAdapter interface {
	Name() string
	SendPrompt(caseID string, step string, prompt string) (json.RawMessage, error)
}

// Identifiable is an optional interface for adapters that can report
// which LLM model ("ghost") is behind the adapter ("shell").
type Identifiable interface {
	Identify() (framework.ModelIdentity, error)
}
