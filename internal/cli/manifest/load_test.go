package manifest

import (
	"path/filepath"
	"testing"
)

func TestResolveSyncConfigBuildsRulesFromRepositories(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 3,
		Repositories: []Repository{
			{
				URL: "https://example.com/base/",
				Files: []RepositoryFile{
					{
						FileName: "a.txt",
						OutDir:   "dest",
						Rename:   "renamed.txt",
						Digest:   "abcdef",
					},
				},
			},
		},
	}

	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 {
		t.Fatalf("expected one file rule, got %d", len(resolved.Files))
	}
	rule := resolved.Files[0]
	if rule.Path != filepath.Join("dest", "renamed.txt") {
		t.Fatalf("unexpected rule path: %s", rule.Path)
	}
	if rule.Checksum != "abcdef" {
		t.Fatalf("unexpected checksum: %s", rule.Checksum)
	}
	if rule.Merge != MergeOverwrite {
		t.Fatalf("unexpected merge: %s", rule.Merge)
	}
	src := resolved.Sources[rule.Source]
	if src.URL != "https://example.com/base/a.txt" {
		t.Fatalf("unexpected source url: %s", src.URL)
	}
}

func TestResolveSyncConfigCollectsAllRepositoryFiles(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 3,
		Repositories: []Repository{
			{
				URL: "https://example.com",
				Files: []RepositoryFile{
					{FileName: "base.txt", OutDir: "."},
					{FileName: "profile.txt", OutDir: "."},
				},
			},
		},
	}
	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(resolved.Files))
	}
}

func TestResolveSyncConfigRejectsUnsupportedRepositoryFields(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 3,
		Repositories: []Repository{
			{
				URL: "https://example.com",
				Files: []RepositoryFile{
					{FileName: "a.txt", OutDir: ".", Encoding: "tar+gzip"},
				},
			},
		},
	}

	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected unsupported field error")
	}
}

func TestValidateSyncConfigRejectsMissingSourceURL(t *testing.T) {
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{
			"s1": {},
		},
		Files: []FileRule{{Source: "s1", Path: "a.txt"}},
	}
	if err := ValidateSyncConfig(cfg); err == nil {
		t.Fatalf("expected missing source url error")
	}
}

func TestResolveSyncConfigRejectsTaskWithoutRunOrDependsOn(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 3,
		Tasks: map[string]TaskDef{
			"broken": {},
		},
	}
	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected validation error for task without run and depends_on")
	}
}
