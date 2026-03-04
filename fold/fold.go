package fold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options configures the fold build.
type Options struct {
	ManifestPath string
	Output       string
	GoFlags      []string
	Verbose      bool
}

// Run loads the manifest, generates main.go, and compiles the binary.
func Run(opts Options) error {
	m, err := LoadManifest(opts.ManifestPath)
	if err != nil {
		return err
	}

	reg := DefaultRegistry()
	src, err := GenerateMain(m, reg)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "origami-fold-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	mainPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainPath, src, 0644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	if schemaFile, ok := m.Bindings["store.schema"]; ok {
		manifestDir := filepath.Dir(opts.ManifestPath)
		srcSchema := filepath.Join(manifestDir, schemaFile)
		data, err := os.ReadFile(srcSchema)
		if err != nil {
			return fmt.Errorf("read store schema %q: %w", srcSchema, err)
		}
		dstSchema := filepath.Join(tmpDir, filepath.Base(schemaFile))
		if err := os.WriteFile(dstSchema, data, 0644); err != nil {
			return fmt.Errorf("write store schema: %w", err)
		}
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "copied store schema: %s → %s\n", srcSchema, dstSchema)
		}
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "generated %s (%d bytes)\n", mainPath, len(src))
		fmt.Fprintf(os.Stderr, "%s\n", string(src))
	}

	output := opts.Output
	if output == "" {
		output = filepath.Join("bin", m.Name)
	}
	if !filepath.IsAbs(output) {
		wd, _ := os.Getwd()
		output = filepath.Join(wd, output)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	goMod, err := findGoMod(m, reg)
	if err != nil {
		return fmt.Errorf("find go.mod: %w", err)
	}

	if err := createBuildModule(tmpDir, m, reg, goMod); err != nil {
		return fmt.Errorf("create build module: %w", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tmpDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	tidy.Env = os.Environ()
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	args := []string{"build", "-o", output}
	args = append(args, opts.GoFlags...)
	args = append(args, ".")

	cmd := exec.Command("go", args...)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "running: go %s (in %s)\n", strings.Join(args, " "), tmpDir)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	fmt.Fprintf(os.Stderr, "built %s\n", output)
	return nil
}

func findGoMod(m *Manifest, reg ModuleRegistry) (string, error) {
	if len(m.Imports) == 0 {
		return "", fmt.Errorf("no imports in manifest")
	}
	goPath, err := reg.ResolveFQCN(m.Imports[0])
	if err != nil {
		return "", err
	}

	parts := strings.Split(goPath, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "/"), nil
	}
	return goPath, nil
}

func createBuildModule(tmpDir string, m *Manifest, reg ModuleRegistry, goMod string) error {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("module %s-fold-build\n\ngo 1.24\n\nrequire (\n", m.Name))

	for _, imp := range m.Imports {
		goPath, err := reg.ResolveFQCN(imp)
		if err != nil {
			return err
		}
		modPath := extractModule(goPath)
		buf.WriteString(fmt.Sprintf("\t%s v0.0.0\n", modPath))
	}
	buf.WriteString(")\n\n")

	for _, imp := range m.Imports {
		goPath, err := reg.ResolveFQCN(imp)
		if err != nil {
			return err
		}
		modPath := extractModule(goPath)
		localPath := findLocalModule(modPath)
		if localPath != "" {
			buf.WriteString(fmt.Sprintf("replace %s => %s\n", modPath, localPath))
		}
	}

	return os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(buf.String()), 0644)
}

func extractModule(goPath string) string {
	parts := strings.Split(goPath, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "/")
	}
	return goPath
}

func findLocalModule(modPath string) string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Workspace", filepath.Base(modPath)),
		filepath.Join(".", filepath.Base(modPath)),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "go.mod")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}
