package fold

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ModuleResolver locates Go modules on the local filesystem.
// The default implementation searches $HOME/Workspace and ./
// but callers can supply custom resolvers for CI or non-standard layouts.
type ModuleResolver interface {
	FindLocalModule(modPath string) string
}

// DefaultModuleResolver searches for Go modules in well-known locations.
type DefaultModuleResolver struct {
	ExtraDirs []string
}

func (r *DefaultModuleResolver) FindLocalModule(modPath string) string {
	home, _ := os.UserHomeDir()
	candidates := make([]string, 0, 2+len(r.ExtraDirs))
	if home != "" {
		candidates = append(candidates, filepath.Join(home, "Workspace", filepath.Base(modPath)))
	}
	candidates = append(candidates, filepath.Join(".", filepath.Base(modPath)))
	for _, d := range r.ExtraDirs {
		candidates = append(candidates, filepath.Join(d, filepath.Base(modPath)))
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "go.mod")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// Options configures the fold build.
type Options struct {
	ManifestPath   string
	Output         string
	GoFlags        []string
	Verbose        bool
	ModuleResolver ModuleResolver
}

// Run loads the manifest, generates the domain-serve source, and compiles the binary.
func Run(opts Options) error {
	m, err := LoadManifest(opts.ManifestPath)
	if err != nil {
		return err
	}

	if m.DomainServe == nil {
		return fmt.Errorf("manifest must have a domain_serve section")
	}

	return buildDomainServe(m, opts)
}

const origamiModule = "github.com/dpopsuev/origami"

func buildDomainServe(m *Manifest, opts Options) error {
	src, err := GenerateDomainServe(m)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "origami-fold-domain-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "domain-serve generated main.go (%d bytes)\n", len(src))
		fmt.Fprintf(os.Stderr, "%s\n", string(src))
	}

	manifestDir := filepath.Dir(opts.ManifestPath)

	if err := copyEmbedFiles(m.DomainServe, manifestDir, tmpDir, opts.Verbose); err != nil {
		return err
	}

	resolver := opts.ModuleResolver
	if resolver == nil {
		resolver = &DefaultModuleResolver{}
	}

	if err := createDomainServeBuildModule(tmpDir, m.Name, resolver); err != nil {
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

	output := opts.Output
	if output == "" {
		output = filepath.Join("bin", m.Name+"-domain-serve")
	}
	if !filepath.IsAbs(output) {
		wd, _ := os.Getwd()
		output = filepath.Join(wd, output)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
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
		return fmt.Errorf("go build domain-serve: %w", err)
	}

	fmt.Fprintf(os.Stderr, "built %s\n", output)
	return nil
}

func createDomainServeBuildModule(tmpDir, name string, resolver ModuleResolver) error {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("module %s-domain-serve-build\n\ngo 1.24\n\nrequire (\n", name))
	buf.WriteString(fmt.Sprintf("\t%s v0.0.0\n", origamiModule))
	buf.WriteString(")\n\n")

	localPath := resolver.FindLocalModule(origamiModule)
	if localPath != "" {
		buf.WriteString(fmt.Sprintf("replace %s => %s\n", origamiModule, localPath))
	}

	return os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(buf.String()), 0644)
}

func copyEmbedFiles(ds *DomainServeConfig, manifestDir, tmpDir string, verbose bool) error {
	if ds.Embed != "" {
		embedDir := strings.TrimRight(ds.Embed, "/")
		srcEmbed := filepath.Join(manifestDir, embedDir)
		dstEmbed := filepath.Join(tmpDir, embedDir)
		if err := copyDir(srcEmbed, dstEmbed); err != nil {
			return fmt.Errorf("copy embed dir %q: %w", embedDir, err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "copied embed dir: %s -> %s\n", srcEmbed, dstEmbed)
		}
		return nil
	}

	paths := ds.Assets.AllPaths()
	for _, p := range paths {
		srcPath := filepath.Join(manifestDir, p)
		dstPath := filepath.Join(tmpDir, p)
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy asset %q: %w", p, err)
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "copied %d asset files\n", len(paths))
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(target)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
