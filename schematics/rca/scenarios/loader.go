package scenarios

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/dpopsuev/origami/schematics/rca"

	"gopkg.in/yaml.v3"
)

// LoadScenario reads a scenario by name from the given filesystem.
// Name is derived from the filename; a bare name: field in the YAML is optional.
func LoadScenario(fsys fs.FS, name string) (*rca.Scenario, error) {
	data, err := fs.ReadFile(fsys, name+".yaml")
	if err != nil {
		return nil, fmt.Errorf("scenario %q not found (available: %s): %w",
			name, strings.Join(ListScenarios(fsys), ", "), err)
	}
	var s rca.Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario %q: %w", name, err)
	}
	if s.Name == "" {
		s.Name = name
	}
	s.ApplyDefaults()
	return &s, nil
}

// ListScenarios returns the names of all scenarios in the given filesystem, sorted.
func ListScenarios(fsys fs.FS) []string {
	entries, _ := fs.ReadDir(fsys, ".")
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(names)
	return names
}
