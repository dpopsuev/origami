package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/dpopsuev/origami/calibrate"
	"github.com/dpopsuev/origami/connectors/docs"
	"github.com/dpopsuev/origami/connectors/github"
	"github.com/dpopsuev/origami/schematics/harvester"
)

func captureCmd(args []string) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	schematic := fs.String("schematic", "", "schematic name (e.g. harvester)")
	sourcePack := fs.String("source-pack", "", "path to source pack YAML")
	output := fs.String("output", "", "output directory for the bundle")
	overwrite := fs.Bool("overwrite", false, "overwrite existing bundle")
	verbose := fs.Bool("v", false, "verbose output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *schematic == "" || *sourcePack == "" || *output == "" {
		return fmt.Errorf("usage: origami capture --schematic=<name> --source-pack=<path> --output=<dir>")
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	cap, err := buildCapturer(*schematic, logger)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	logger.Info("starting capture", "schematic", *schematic, "source_pack", *sourcePack, "output", *output)

	cfg := calibrate.CaptureConfig{
		Schematic:  *schematic,
		SourcePack: *sourcePack,
		OutputDir:  *output,
		Overwrite:  *overwrite,
	}

	if err := cap.Capture(ctx, cfg); err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	logger.Info("capture complete", "output", *output)
	return nil
}

func buildCapturer(schematic string, logger *slog.Logger) (calibrate.Capturer, error) {
	switch schematic {
	case "harvester":
		gitDrv, err := github.DefaultGitDriver()
		if err != nil {
			return nil, fmt.Errorf("init git driver: %w", err)
		}
		docsDrv := docs.NewDocsDriver()
		router := harvester.NewRouter(
			harvester.WithGitDriver(gitDrv),
			harvester.WithDocsDriver(docsDrv),
		)
		return harvester.NewCapturer(router, logger), nil
	default:
		return nil, fmt.Errorf("unknown schematic %q (available: harvester)", schematic)
	}
}
