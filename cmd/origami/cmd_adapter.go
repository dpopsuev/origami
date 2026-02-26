package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	framework "github.com/dpopsuev/origami"
)

func adapterCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: origami adapter <list|inspect|validate> [flags]")
	}
	switch args[0] {
	case "list":
		return adapterList(args[1:])
	case "inspect":
		return adapterInspect(args[1:])
	case "validate":
		return adapterValidate(args[1:])
	default:
		return fmt.Errorf("unknown adapter subcommand: %s", args[0])
	}
}

func adapterList(args []string) error {
	fs := flag.NewFlagSet("adapter list", flag.ContinueOnError)
	dir := fs.String("dir", ".", "directory to scan for adapter.yaml files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var manifests []*framework.AdapterManifest
	filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "adapter.yaml" {
			m, loadErr := framework.LoadAdapterManifest(path)
			if loadErr == nil {
				manifests = append(manifests, m)
			}
		}
		return nil
	})

	if len(manifests) == 0 {
		fmt.Println("No adapters found.")
		return nil
	}

	fmt.Printf("%-20s %-10s %-12s %s\n", "NAMESPACE", "VERSION", "ADAPTER", "PROVIDES")
	for _, m := range manifests {
		provides := make([]string, 0)
		if len(m.Provides.Transformers) > 0 {
			provides = append(provides, fmt.Sprintf("T:%s", strings.Join(m.Provides.Transformers, ",")))
		}
		if len(m.Provides.Extractors) > 0 {
			provides = append(provides, fmt.Sprintf("E:%s", strings.Join(m.Provides.Extractors, ",")))
		}
		if len(m.Provides.Hooks) > 0 {
			provides = append(provides, fmt.Sprintf("H:%s", strings.Join(m.Provides.Hooks, ",")))
		}
		fmt.Printf("%-20s %-10s %-12s %s\n", m.Namespace, m.Version, m.Adapter, strings.Join(provides, " "))
	}
	return nil
}

func adapterInspect(args []string) error {
	fs := flag.NewFlagSet("adapter inspect", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: origami adapter inspect <adapter.yaml>")
	}

	m, err := framework.LoadAdapterManifest(fs.Arg(0))
	if err != nil {
		return err
	}

	fmt.Printf("Adapter:     %s\n", m.Adapter)
	fmt.Printf("Namespace:   %s\n", m.Namespace)
	fmt.Printf("Version:     %s\n", m.Version)
	if m.Description != "" {
		fmt.Printf("Description: %s\n", m.Description)
	}
	if m.Requires.Origami != "" {
		fmt.Printf("Requires:    origami %s\n", m.Requires.Origami)
	}
	if len(m.Provides.Transformers) > 0 {
		fmt.Printf("Transformers: %s\n", strings.Join(m.Provides.Transformers, ", "))
	}
	if len(m.Provides.Extractors) > 0 {
		fmt.Printf("Extractors:   %s\n", strings.Join(m.Provides.Extractors, ", "))
	}
	if len(m.Provides.Hooks) > 0 {
		fmt.Printf("Hooks:        %s\n", strings.Join(m.Provides.Hooks, ", "))
	}
	return nil
}

func adapterValidate(args []string) error {
	fs := flag.NewFlagSet("adapter validate", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: origami adapter validate <adapter.yaml>")
	}

	path := fs.Arg(0)
	m, err := framework.LoadAdapterManifest(path)
	if err != nil {
		return err
	}

	var issues []string
	if m.Adapter == "" {
		issues = append(issues, "missing adapter name")
	}
	if m.Namespace == "" {
		issues = append(issues, "missing namespace")
	}
	if m.Version == "" {
		issues = append(issues, "missing version")
	}
	total := len(m.Provides.Transformers) + len(m.Provides.Extractors) + len(m.Provides.Hooks)
	if total == 0 {
		issues = append(issues, "provides section is empty")
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "  ✗ %s\n", issue)
		}
		return fmt.Errorf("adapter manifest %s has %d issue(s)", path, len(issues))
	}

	fmt.Printf("OK: %s (%s/%s) is valid\n", path, m.Namespace, m.Adapter)
	return nil
}
