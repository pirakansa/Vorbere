package manifest

import (
	"os"

	"gopkg.in/yaml.v3"
)

const LockFileName = "vorbere.lock"

// LockFile tracks last applied sync state for conflict detection.
type LockFile struct {
	Version string               `yaml:"version"`
	Files   map[string]LockEntry `yaml:"files"`
}

// LockEntry stores file-level sync metadata.
type LockEntry struct {
	SourceURL   string `yaml:"source_url"`
	AppliedHash string `yaml:"applied_hash"`
	SourceHash  string `yaml:"source_hash"`
	UpdatedAt   string `yaml:"updated_at"`
}

func LoadLock(path string) (*LockFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LockFile{Version: "v1", Files: map[string]LockEntry{}}, nil
		}
		return nil, err
	}
	var lock LockFile
	if err := yaml.Unmarshal(b, &lock); err != nil {
		return nil, err
	}
	if lock.Version == "" {
		lock.Version = "v1"
	}
	if lock.Files == nil {
		lock.Files = map[string]LockEntry{}
	}
	return &lock, nil
}

func SaveLock(path string, lock *LockFile) error {
	if lock.Version == "" {
		lock.Version = "v1"
	}
	if lock.Files == nil {
		lock.Files = map[string]LockEntry{}
	}
	b, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
