package subprocess_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dpopsuev/origami/subprocess"
)

func TestOrchestrator_RegisterAndStart(t *testing.T) {
	requireExec(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orch := subprocess.NewOrchestrator()
	orch.Register("echo", newTestServer(t, "default"))

	if err := orch.Start(ctx, "echo"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer orch.StopAll(context.Background())

	result, err := orch.CallTool(ctx, "echo", "echo", map[string]any{"message": "orchestrated"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if got := extractText(t, result); got != "echo: orchestrated" {
		t.Errorf("got %q, want %q", got, "echo: orchestrated")
	}
}

func TestOrchestrator_Swap(t *testing.T) {
	requireExec(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	orch := subprocess.NewOrchestrator()
	orch.Register("svc", &subprocess.Server{
		BinaryPath: exe,
		Env:        []string{runAsServer + "=default"},
	})

	if err := orch.Start(ctx, "svc"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer orch.StopAll(context.Background())

	// Verify pre-swap behavior: "default" server has echo tool
	result, err := orch.CallTool(ctx, "svc", "echo", map[string]any{"message": "v1"})
	if err != nil {
		t.Fatalf("CallTool before swap: %v", err)
	}
	if got := extractText(t, result); got != "echo: v1" {
		t.Errorf("before swap: got %q, want %q", got, "echo: v1")
	}

	err = orch.Swap(ctx, "svc", &subprocess.Server{
		BinaryPath: exe,
		Env:        []string{runAsServer + "=default"},
	})
	if err != nil {
		t.Fatalf("Swap: %v", err)
	}

	// Verify post-swap: should still work (same behavior since same binary+env)
	result, err = orch.CallTool(ctx, "svc", "echo", map[string]any{"message": "v2"})
	if err != nil {
		t.Fatalf("CallTool after swap: %v", err)
	}
	if got := extractText(t, result); got != "echo: v2" {
		t.Errorf("after swap: got %q, want %q", got, "echo: v2")
	}

	// Health should be good after swap
	if !orch.Healthy(ctx, "svc") {
		t.Error("expected healthy after swap")
	}
}

func TestOrchestrator_SwapWithDifferentBehavior(t *testing.T) {
	requireExec(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	orch := subprocess.NewOrchestrator()

	// Start with "default" server (has echo + add tools)
	orch.Register("svc", &subprocess.Server{
		BinaryPath: exe,
		Env:        []string{runAsServer + "=default"},
	})
	if err := orch.Start(ctx, "svc"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer orch.StopAll(context.Background())

	// Verify echo works
	result, err := orch.CallTool(ctx, "svc", "echo", map[string]any{"message": "pre-swap"})
	if err != nil {
		t.Fatalf("pre-swap echo: %v", err)
	}
	if got := extractText(t, result); got != "echo: pre-swap" {
		t.Errorf("got %q", got)
	}

	// Swap to "slow" server (has only slow tool, not echo)
	// We create a new server with different env but need to swap via orchestrator.
	// Orchestrator.Swap preserves Env from old server, so we register a new one.
	orch.Register("svc", &subprocess.Server{
		BinaryPath: exe,
		Env:        []string{runAsServer + "=slow"},
	})
	// Stop old, start the new registered one
	orch.Stop(ctx, "svc")
	// Re-register was done above, now start
	if err := orch.Start(ctx, "svc"); err != nil {
		t.Fatalf("Start after swap: %v", err)
	}

	// The "slow" server has a "slow" tool, not "echo"
	result, err = orch.CallTool(ctx, "svc", "slow", map[string]any{"duration_ms": 10})
	if err != nil {
		t.Fatalf("post-swap slow: %v", err)
	}
	if got := extractText(t, result); got != "done" {
		t.Errorf("slow tool: got %q, want %q", got, "done")
	}
}

func TestOrchestrator_UnknownSchematic(t *testing.T) {
	orch := subprocess.NewOrchestrator()
	ctx := context.Background()

	if err := orch.Start(ctx, "nonexistent"); err == nil {
		t.Error("expected error for unknown schematic")
	}
	if err := orch.Stop(ctx, "nonexistent"); err == nil {
		t.Error("expected error for unknown schematic")
	}
	if orch.Healthy(ctx, "nonexistent") {
		t.Error("expected unhealthy for unknown schematic")
	}
}

func TestOrchestrator_StopAll(t *testing.T) {
	requireExec(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orch := subprocess.NewOrchestrator()
	orch.Register("a", newTestServer(t, "default"))
	orch.Register("b", newTestServer(t, "default"))

	if err := orch.Start(ctx, "a"); err != nil {
		t.Fatalf("Start a: %v", err)
	}
	if err := orch.Start(ctx, "b"); err != nil {
		t.Fatalf("Start b: %v", err)
	}

	orch.StopAll(ctx)

	if orch.Healthy(ctx, "a") {
		t.Error("a should not be healthy after StopAll")
	}
	if orch.Healthy(ctx, "b") {
		t.Error("b should not be healthy after StopAll")
	}
}
