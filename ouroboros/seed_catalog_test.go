package ouroboros

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeedCatalog_AllSeedsValid(t *testing.T) {
	seedDir := "seeds"
	entries, err := os.ReadDir(seedDir)
	if err != nil {
		t.Fatalf("read seeds directory: %v", err)
	}

	if len(entries) < 40 {
		t.Fatalf("expected at least 40 seeds, found %d", len(entries))
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(seedDir, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			seed, err := LoadSeed(path)
			if err != nil {
				t.Fatalf("load failed: %v", err)
			}
			if len(seed.Poles) != 2 {
				t.Errorf("poles = %d, want 2", len(seed.Poles))
			}
			if seed.Context == "" {
				t.Error("context is empty")
			}
			if seed.Rubric == "" {
				t.Error("rubric is empty")
			}
			for _, dim := range seed.Dimensions {
				if !knownDimensions[dim] {
					t.Errorf("unknown dimension: %q", dim)
				}
			}
		})
	}
}

func TestSeedCatalog_CoversDimensions(t *testing.T) {
	seedDir := "seeds"
	entries, err := os.ReadDir(seedDir)
	if err != nil {
		t.Fatalf("read seeds directory: %v", err)
	}

	covered := make(map[Dimension]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		seed, err := LoadSeed(filepath.Join(seedDir, entry.Name()))
		if err != nil {
			continue
		}
		for _, dim := range seed.Dimensions {
			covered[dim] = true
		}
	}

	for _, dim := range AllDimensions() {
		if !covered[dim] {
			t.Errorf("dimension %q not covered by any seed", dim)
		}
	}
}

func TestSeedCatalog_CoversCategories(t *testing.T) {
	seedDir := "seeds"
	entries, err := os.ReadDir(seedDir)
	if err != nil {
		t.Fatalf("read seeds directory: %v", err)
	}

	covered := make(map[SeedCategory]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		seed, err := LoadSeed(filepath.Join(seedDir, entry.Name()))
		if err != nil {
			continue
		}
		covered[seed.Category] = true
	}

	for _, cat := range []SeedCategory{CategorySkill, CategoryTrap, CategoryBoundary, CategoryIdentity, CategoryReframe} {
		if !covered[cat] {
			t.Errorf("category %q not covered by any seed", cat)
		}
	}
}
