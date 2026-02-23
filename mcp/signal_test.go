package mcp

import (
	"testing"
)

// TestSignalBus_Alias verifies the backward-compat aliases delegate correctly.
func TestSignalBus_Alias(t *testing.T) {
	bus := NewSignalBus()
	bus.Emit("alias_test", "agent", "C1", "F0", map[string]string{"via": "mcp_alias"})
	if bus.Len() != 1 {
		t.Fatalf("alias bus Len: got %d, want 1", bus.Len())
	}
	sigs := bus.Since(0)
	if sigs[0].Event != "alias_test" {
		t.Errorf("alias event: got %q, want alias_test", sigs[0].Event)
	}
}
