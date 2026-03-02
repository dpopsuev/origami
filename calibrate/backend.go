package calibrate

import (
	"encoding/json"

	framework "github.com/dpopsuev/origami"
)

// ModelBackend is the interface for sending prompts and receiving responses
// during calibration. The step parameter is a plain string so consumers
// can use their own step type (e.g. CircuitStep) without coupling the
// framework to a domain-specific enum.
type ModelBackend interface {
	Name() string
	SendPrompt(caseID string, step string, prompt string) (json.RawMessage, error)
}

// Identifiable is an optional interface for backends that can report
// which LLM model ("ghost") is behind the backend ("shell").
type Identifiable interface {
	Identify() (framework.ModelIdentity, error)
}
