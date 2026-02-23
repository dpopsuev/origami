package framework

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONCheckpointer persists WalkerState to a JSON file between nodes,
// enabling resume-from-failure for pipelines.
//
// This is a PoC battery — sufficient for prototyping, not production-grade.
// Consumers should replace it with their own checkpointing for production use.
type JSONCheckpointer struct {
	Dir string
}

// NewJSONCheckpointer creates a checkpointer that writes to the given directory.
func NewJSONCheckpointer(dir string) (*JSONCheckpointer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("checkpoint: create dir: %w", err)
	}
	return &JSONCheckpointer{Dir: dir}, nil
}

// Save persists the walker state to a JSON file named by the walker's ID.
func (c *JSONCheckpointer) Save(state *WalkerState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint: marshal state: %w", err)
	}
	path := c.path(state.ID)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("checkpoint: write %s: %w", path, err)
	}
	return nil
}

// Load restores a walker state from a previously saved checkpoint file.
// Returns nil and no error if no checkpoint exists for the given ID.
func (c *JSONCheckpointer) Load(id string) (*WalkerState, error) {
	path := c.path(id)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("checkpoint: read %s: %w", path, err)
	}
	var state WalkerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("checkpoint: unmarshal %s: %w", path, err)
	}
	if state.LoopCounts == nil {
		state.LoopCounts = make(map[string]int)
	}
	if state.Context == nil {
		state.Context = make(map[string]any)
	}
	if state.Outputs == nil {
		state.Outputs = make(map[string]Artifact)
	}
	return &state, nil
}

// Remove deletes the checkpoint file for the given walker ID.
func (c *JSONCheckpointer) Remove(id string) error {
	path := c.path(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checkpoint: remove %s: %w", path, err)
	}
	return nil
}

func (c *JSONCheckpointer) path(id string) string {
	return filepath.Join(c.Dir, id+".checkpoint.json")
}
