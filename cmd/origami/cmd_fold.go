package main

import (
	"flag"
	"fmt"

	"github.com/dpopsuev/origami/fold"
)

func foldCmd(args []string) error {
	fs := flag.NewFlagSet("fold", flag.ContinueOnError)
	output := fs.String("output", "", "output binary path (default: bin/<name>-domain-serve)")
	container := fs.Bool("container", false, "build an OCI container image after compiling")
	imageName := fs.String("image", "", "container image name (default: origami-<name>-domain)")
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
		Container:    *container,
		ImageName:    *imageName,
		Verbose:      *verbose,
	})
}

func foldUsage() string {
	return fmt.Sprintf("  fold       Compile a YAML manifest into a standalone binary")
}
