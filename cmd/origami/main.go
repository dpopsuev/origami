package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	framework "github.com/dpopsuev/origami"
	"github.com/dpopsuev/origami/transformers"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if err := runCmd(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := validateCmd(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Println("origami v1.0.0")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: origami <command> [flags]

Commands:
  run        Execute a pipeline YAML
  validate   Validate a pipeline YAML without executing
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
