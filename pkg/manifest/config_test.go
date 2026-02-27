package manifest

import (
	"path/filepath"
	"testing"
)

func TestBuildSyncConfigBuildsRulesFromRepositories(t *testing.T) {
	cfg := &TaskConfig{
		Version: 3,
		Repositories: []Repository{{
			URL: "https://example.com/base/",
			Files: []RepositoryFile{{
				FileName: "a.txt",
				OutDir:   "dest",
				Rename:   "renamed.txt",
				Digest:   "abcdef",
			}},
		}},
	}

	resolved, err := BuildSyncConfig(cfg)
	if err != nil {
		t.Fatalf("BuildSyncConfig returned error: %v", err)
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
