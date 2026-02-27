package manifest

import (
	"net/http"
	"net/http/httptest"
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

func TestResolveSyncConfigHTTPRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`version: v1
sources:
  s1:
    type: http
    url: https://example.com/managed
files:
  - source: s1
    path: managed.txt
`))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: "v1",
		Sync: SyncRef{
			Ref: server.URL,
		},
	}

	resolved, err := ResolveSyncConfig(cfg, filepath.Join(temp, "task.yaml"))
	if err != nil {
		t.Fatalf("ResolveSyncConfig returned error: %v", err)
	}
	if len(resolved.Files) != 1 || resolved.Files[0].Path != "managed.txt" {
		t.Fatalf("unexpected resolved HTTP ref config: %#v", resolved)
	}
}

func TestResolveProfileFilesAppendsProfileEntries(t *testing.T) {
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{
			"base": {Type: "http", URL: "https://example.com/base"},
			"dev":  {Type: "http", URL: "https://example.com/dev"},
		},
		Files: []FileRule{
			{Source: "base", Path: "a.txt"},
		},
		Profiles: map[string]Profile{
			"devcontainer": {
				Files: []FileRule{
					{Source: "dev", Path: "b.txt"},
				},
			},
		},
	}

	files, err := ResolveProfileFiles(cfg, "devcontainer")
	if err != nil {
		t.Fatalf("ResolveProfileFiles returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files got %d", len(files))
	}
	if files[0].Path != "a.txt" || files[1].Path != "b.txt" {
		t.Fatalf("unexpected file order/content: %#v", files)
	}
}

func TestValidateSyncConfigRejectsUnsupportedSourceType(t *testing.T) {
	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{
			"s1": {Type: "git", URL: "https://example.com/repo.git"},
		},
		Files: []FileRule{{Source: "s1", Path: "a.txt"}},
	}
	if err := ValidateSyncConfig(cfg); err == nil {
		t.Fatalf("expected unsupported source type error")
	}
}

func TestResolveSyncConfigRejectsTaskWithoutRunOrDependsOn(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: "v1",
		Tasks: map[string]TaskDef{
			"broken": {},
		},
	}
	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "task.yaml")); err == nil {
		t.Fatalf("expected validation error for task without run and depends_on")
	}
}
