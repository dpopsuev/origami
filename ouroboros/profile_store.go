package ouroboros

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProfileStore persists and retrieves Ouroboros model profiles.
// Implementations must be append-only — never overwrite or delete a profile.
type ProfileStore interface {
	Save(profile ModelProfile) error
	Load(id string) (ModelProfile, error)
	List() ([]string, error)
	History(modelName string) ([]ModelProfile, error)
}

// FileProfileStore implements ProfileStore using one JSON file per profile.
// Files are keyed by model name + timestamp to guarantee uniqueness and
// preserve the full calibration history.
type FileProfileStore struct {
	Dir string
}

// NewFileProfileStore creates a FileProfileStore, ensuring the directory exists.
func NewFileProfileStore(dir string) (*FileProfileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create profile store dir: %w", err)
	}
	return &FileProfileStore{Dir: dir}, nil
}

// Save writes a model profile as pretty-printed JSON. The filename is
// derived from the model name and timestamp. Returns an error if a file
// with that key already exists (append-only guarantee).
func (s *FileProfileStore) Save(profile ModelProfile) error {
	id := profileID(profile)
	path := s.pathFor(id)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("profile %q already exists at %s (append-only: refusing to overwrite)", id, path)
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}

	return nil
}

// Load reads a profile by its ID (model-name_timestamp).
func (s *FileProfileStore) Load(id string) (ModelProfile, error) {
	data, err := os.ReadFile(s.pathFor(id))
	if err != nil {
		return ModelProfile{}, fmt.Errorf("read profile %q: %w", id, err)
	}

	var profile ModelProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return ModelProfile{}, fmt.Errorf("parse profile %q: %w", id, err)
	}

	return profile, nil
}

// List returns all profile IDs in the store, sorted alphabetically.
func (s *FileProfileStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(ids)
	return ids, nil
}

// History returns all profiles for a given model name, sorted by timestamp
// (oldest first). This enables tracking behavioral evolution across versions.
func (s *FileProfileStore) History(modelName string) ([]ModelProfile, error) {
	prefix := sanitizeName(modelName) + "_"

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("list profiles for history: %w", err)
	}

	var profiles []ModelProfile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		p, err := s.Load(id)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Timestamp.Before(profiles[j].Timestamp)
	})

	return profiles, nil
}

func (s *FileProfileStore) pathFor(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

func profileID(p ModelProfile) string {
	ts := p.Timestamp.UTC().Format(time.RFC3339)
	ts = strings.ReplaceAll(ts, ":", "-")
	return sanitizeName(p.Model.ModelName) + "_" + ts
}

func sanitizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, strings.ToLower(name))
}
