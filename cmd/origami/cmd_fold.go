package main

import (
	"flag"
	"fmt"

	"github.com/dpopsuev/origami/fold"
)

func foldCmd(args []string) error {
	fs := flag.NewFlagSet("fold", flag.ContinueOnError)
	output := fs.String("output", "", "output binary path (default: bin/<name>)")
	verbose := fs.Bool("v", false, "verbose output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	manifest := "origami.yaml"
	if fs.NArg() > 0 {
		manifest = fs.Arg(0)
	}

	return fold.Run(fold.Options{
		ManifestPath: manifest,
		Output:       *output,
		Verbose:      *verbose,
	})
}

func foldUsage() string {
	return fmt.Sprintf("  fold       Compile a YAML manifest into a standalone binary")
}
