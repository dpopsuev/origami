package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	framework "github.com/dpopsuev/origami"
	fwmcp "github.com/dpopsuev/origami/mcp"
	"github.com/dpopsuev/origami/ouroboros"
	"github.com/dpopsuev/origami/ouroborosmcp"
	"github.com/dpopsuev/origami/transformers"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = runCmd(os.Args[2:])
	case "validate":
		err = validateCmd(os.Args[2:])
	case "ouroboros":
		err = ouroborosCmd(os.Args[2:])
	case "version":
		fmt.Println("origami v1.0.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: origami <command> [flags]

Commands:
  run        Execute a pipeline YAML
  validate   Validate a pipeline YAML without executing
  ouroboros  Ouroboros meta-calibration tools (prompt, analyze, save, serve)
  version    Print version`)
}

type setFlag map[string]any

func (s setFlag) String() string { return fmt.Sprintf("%v", map[string]any(s)) }
func (s setFlag) Set(v string) error {
	parts := strings.SplitN(v, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected key=value, got %q", v)
	}
	s[parts[0]] = parts[1]
	return nil
}

func runCmd(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose output (debug level)")
	sets := make(setFlag)
	fs.Var(sets, "set", "set pipeline variable (key=value), repeatable")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: origami run [-v] [--set key=value] <pipeline.yaml>")
	}
	pipelinePath := fs.Arg(0)

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	builtins := framework.TransformerRegistry{
		"file": transformers.NewFile(transformers.WithRootDir(filepath.Dir(pipelinePath))),
	}

	opts := []framework.RunOption{
		framework.WithLogger(logger),
		framework.WithTransformers(builtins),
	}
	if len(sets) > 0 {
		opts = append(opts, framework.WithOverrides(map[string]any(sets)))
	}

	logger.Info("running pipeline", "path", pipelinePath)
	if err := framework.Run(ctx, pipelinePath, nil, opts...); err != nil {
		return err
	}
	logger.Info("pipeline completed")
	return nil
}

func validateCmd(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: origami validate <pipeline.yaml>")
	}
	pipelinePath := fs.Arg(0)

	if err := framework.Validate(pipelinePath); err != nil {
		return err
	}
	fmt.Printf("OK: %s is valid\n", pipelinePath)
	return nil
}

// --- ouroboros subcommand group ---

func ouroborosCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: origami ouroboros <prompt|analyze|save|serve> [flags]")
	}
	switch args[0] {
	case "prompt":
		return ouroborosPrompt(args[1:])
	case "analyze":
		return ouroborosAnalyze(args[1:])
	case "save":
		return ouroborosSave(args[1:])
	case "serve":
		return ouroborosServe(args[1:])
	default:
		return fmt.Errorf("unknown ouroboros subcommand: %s", args[0])
	}
}

func ouroborosPrompt(args []string) error {
	fs := flag.NewFlagSet("ouroboros prompt", flag.ContinueOnError)
	excludeFile := fs.String("exclude-file", "", "JSON file with array of ModelIdentity to exclude")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var exclude []framework.ModelIdentity
	if *excludeFile != "" {
		data, err := os.ReadFile(*excludeFile)
		if err != nil {
			return fmt.Errorf("read exclude file: %w", err)
		}
		if err := json.Unmarshal(data, &exclude); err != nil {
			return fmt.Errorf("parse exclude file: %w", err)
		}
	}

	fmt.Print(ouroboros.BuildFullPrompt(exclude))
	return nil
}

type analyzeResult struct {
	Identity framework.ModelIdentity `json:"identity"`
	Key      string                  `json:"key"`
	Code     string                  `json:"code"`
	Score    ouroboros.ProbeScore      `json:"score"`
	Known    bool                    `json:"known"`
	Wrapper  bool                    `json:"wrapper"`
}

func ouroborosAnalyze(args []string) error {
	fs := flag.NewFlagSet("ouroboros analyze", flag.ContinueOnError)
	responseFile := fs.String("response-file", "", "text file with raw subagent response (- for stdin)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *responseFile == "" {
		return fmt.Errorf("--response-file is required")
	}

	var data []byte
	var err error
	if *responseFile == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*responseFile)
	}
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	raw := string(data)

	mi, err := ouroboros.ParseIdentityResponse(raw)
	if err != nil {
		return fmt.Errorf("parse identity: %w", err)
	}

	code, err := ouroboros.ParseProbeResponse(raw)
	if err != nil {
		return fmt.Errorf("parse code: %w", err)
	}

	score := ouroboros.ScoreRefactorOutput(code)

	result := analyzeResult{
		Identity: mi,
		Key:      ouroboros.ModelKey(mi),
		Code:     code,
		Score:    score,
		Known:    framework.IsKnownModel(mi),
		Wrapper:  framework.IsWrapperName(mi.ModelName),
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

const defaultRunsDir = "ouroboros/runs"

func ouroborosSave(args []string) error {
	fs := flag.NewFlagSet("ouroboros save", flag.ContinueOnError)
	reportFile := fs.String("report-file", "", "JSON file containing the RunReport (- for stdin)")
	runsDir := fs.String("runs-dir", defaultRunsDir, "directory to save run reports")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *reportFile == "" {
		return fmt.Errorf("--report-file is required")
	}

	var data []byte
	var err error
	if *reportFile == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*reportFile)
	}
	if err != nil {
		return fmt.Errorf("read report: %w", err)
	}

	var report ouroboros.RunReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("parse report: %w", err)
	}

	store, err := ouroboros.NewFileRunStore(*runsDir)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	if err := store.SaveRun(report); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "saved run %q to %s\n", report.RunID, *runsDir)
	return nil
}

func ouroborosServe(args []string) error {
	fs := flag.NewFlagSet("ouroboros serve", flag.ContinueOnError)
	runsDir := fs.String("runs-dir", defaultRunsDir, "directory to save discovery run reports")
	if err := fs.Parse(args); err != nil {
		return err
	}

	srv := ouroborosmcp.NewServer(*runsDir)
	defer srv.Shutdown()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fwmcp.WatchStdin(ctx, nil, cancel)

	slog.Info("starting ouroboros MCP server over stdio", "runs_dir", *runsDir)
	return srv.MCPServer.Run(ctx, &sdkmcp.StdioTransport{})
}
