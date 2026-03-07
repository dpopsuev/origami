package probes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpopsuev/origami/ouroboros"
)

// VerifyResult holds the outcome of executing a code-producing probe's output.
type VerifyResult struct {
	Compiled        bool   `json:"compiled"`
	TestsPassed     bool   `json:"tests_passed"`
	BenchmarkPassed bool   `json:"benchmark_passed,omitempty"`
	BenchmarkMs     int    `json:"benchmark_ms,omitempty"`
	CompileErr      string `json:"compile_error,omitempty"`
	TestErr         string `json:"test_error,omitempty"`
	BenchmarkErr    string `json:"benchmark_error,omitempty"`
}

// VerifyCode extracts code from a subject response, writes it to a temp dir
// alongside the seed's test file, and runs compile + test + benchmark.
// Returns a binary VerifyResult. Timeout: 30s per command.
func VerifyCode(raw string, verify *ouroboros.SeedVerify) (*VerifyResult, error) {
	if verify == nil {
		return nil, fmt.Errorf("verify: no verification config")
	}

	code := ExtractCodeBlock(raw)
	if code == "" {
		return &VerifyResult{
			Compiled: false, CompileErr: "no code block found in response",
		}, nil
	}

	dir, err := os.MkdirTemp("", "ouroboros-verify-*")
	if err != nil {
		return nil, fmt.Errorf("verify: create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	ext := languageExtension(verify.Language)
	codePath := filepath.Join(dir, "solution"+ext)
	if err := os.WriteFile(codePath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("verify: write code: %w", err)
	}

	if verify.TestFile != "" {
		testPath := filepath.Join(dir, "solution_test"+ext)
		if err := os.WriteFile(testPath, []byte(verify.TestFile), 0644); err != nil {
			return nil, fmt.Errorf("verify: write test file: %w", err)
		}
	}

	if verify.Language == "go" || verify.Language == "Go" {
		modContent := "module verify\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modContent), 0644); err != nil {
			return nil, fmt.Errorf("verify: write go.mod: %w", err)
		}
	}

	if verify.SetupCommands != "" {
		if _, err := runCommand(dir, verify.SetupCommands); err != nil {
			return nil, fmt.Errorf("verify: setup commands: %w", err)
		}
	}

	result := &VerifyResult{}

	if verify.Compile != "" {
		out, err := runCommand(dir, verify.Compile)
		if err != nil {
			result.Compiled = false
			result.CompileErr = fmt.Sprintf("%s: %s", err, out)
			return result, nil
		}
		result.Compiled = true
	} else {
		result.Compiled = true
	}

	if verify.Test != "" {
		out, err := runCommand(dir, verify.Test)
		if err != nil {
			result.TestsPassed = false
			result.TestErr = fmt.Sprintf("%s: %s", err, out)
			return result, nil
		}
		result.TestsPassed = true
	} else {
		result.TestsPassed = result.Compiled
	}

	if verify.Benchmark != "" && result.TestsPassed {
		start := time.Now()
		out, err := runCommand(dir, verify.Benchmark)
		elapsed := time.Since(start)
		result.BenchmarkMs = int(elapsed.Milliseconds())

		if err != nil {
			result.BenchmarkPassed = false
			result.BenchmarkErr = fmt.Sprintf("%s: %s", err, out)
		} else if verify.BenchmarkThreshMs > 0 && result.BenchmarkMs > verify.BenchmarkThreshMs {
			result.BenchmarkPassed = false
			result.BenchmarkErr = fmt.Sprintf("exceeded threshold: %dms > %dms", result.BenchmarkMs, verify.BenchmarkThreshMs)
		} else {
			result.BenchmarkPassed = true
		}
	}

	return result, nil
}

// ToMechanical converts a VerifyResult into the domain-level MechanicalVerifyResult
// that lives on PoleResult for observability and downstream scoring.
func ToMechanical(vr *VerifyResult) *ouroboros.MechanicalVerifyResult {
	score := 0.0
	if vr.Compiled {
		score += 0.3
	}
	if vr.TestsPassed {
		score += 0.4
	}
	if vr.BenchmarkPassed {
		score += 0.3
	}

	return &ouroboros.MechanicalVerifyResult{
		Compiled:        vr.Compiled,
		TestsPassed:     vr.TestsPassed,
		BenchmarkPassed: vr.BenchmarkPassed,
		BenchmarkMs:     vr.BenchmarkMs,
		CompileErr:      vr.CompileErr,
		TestErr:         vr.TestErr,
		BenchmarkErr:    vr.BenchmarkErr,
		Score:           clamp(score),
	}
}

// ScoreVerifyResult converts a VerifyResult into dimension adjustments.
// Compilation: +0.3 evidence_depth. Tests: +0.4 evidence_depth.
// Benchmark: +0.3 evidence_depth. Also penalizes speed dimension for
// code that doesn't compile (fast but wrong = bad).
func ScoreVerifyResult(vr *VerifyResult) map[ouroboros.Dimension]float64 {
	depth := 0.0
	if vr.Compiled {
		depth += 0.3
	}
	if vr.TestsPassed {
		depth += 0.4
	}
	if vr.BenchmarkPassed {
		depth += 0.3
	}

	scores := map[ouroboros.Dimension]float64{
		ouroboros.DimEvidenceDepth: clamp(depth),
	}

	if !vr.Compiled {
		scores[ouroboros.DimShortcutAffinity] = 0.9
	}

	return scores
}

// MergeVerifyScores folds mechanical verification results into the Judge's
// dimension scores. Verify-sourced scores override the LLM-classified scores
// for the affected dimensions.
func MergeVerifyScores(dimScores map[ouroboros.Dimension]float64, verifyScores map[ouroboros.Dimension]float64) {
	for dim, score := range verifyScores {
		existing, ok := dimScores[dim]
		if !ok {
			dimScores[dim] = score
		} else {
			dimScores[dim] = (existing + score) / 2
		}
	}
}

// ExtractCodeBlock finds the first fenced code block in a response.
// Falls back to returning the full input if no fences are found.
func ExtractCodeBlock(raw string) string {
	lines := strings.Split(raw, "\n")
	var inBlock bool
	var code strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if inBlock {
				return code.String()
			}
			inBlock = true
			continue
		}
		if inBlock {
			code.WriteString(line)
			code.WriteByte('\n')
		}
	}
	if code.Len() > 0 {
		return code.String()
	}
	return raw
}

func languageExtension(lang string) string {
	switch strings.ToLower(lang) {
	case "go":
		return ".go"
	case "python", "py":
		return ".py"
	case "rust", "rs":
		return ".rs"
	case "typescript", "ts":
		return ".ts"
	default:
		return ".txt"
	}
}

func runCommand(dir, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return string(out), err
}
