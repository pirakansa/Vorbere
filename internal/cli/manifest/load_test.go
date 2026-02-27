package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSyncConfigInlinePreferred(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: "v1",
		Sync: SyncRef{
			Ref: "sync.yaml",
			Inline: &SyncConfig{
				Version: "v1",
				Sources: map[string]Source{"a": {Type: "http", URL: "https://example.com"}},
				Files:   []FileRule{{Source: "a", Path: "a.txt"}},
			},
		},
	}
	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "task.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 || resolved.Files[0].Path != "a.txt" {
		t.Fatalf("unexpected resolved inline config: %#v", resolved)
	}
}

func TestResolveSyncConfigFallbackFile(t *testing.T) {
	temp := t.TempDir()
	syncPath := filepath.Join(temp, "sync.yaml")
	syncBody := `version: v1
sources:
  s1:
    type: http
    url: https://example.com/a
files:
  - source: s1
    path: dest/a.txt
`
	if err := os.WriteFile(syncPath, []byte(syncBody), 0o644); err != nil {
		t.Fatalf("write sync.yaml: %v", err)
	}

	cfg := &TaskConfig{Version: "v1", Tasks: map[string]TaskDef{}}
	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "task.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 || resolved.Files[0].Path != "dest/a.txt" {
		t.Fatalf("unexpected resolved fallback config: %#v", resolved)
	}
}
