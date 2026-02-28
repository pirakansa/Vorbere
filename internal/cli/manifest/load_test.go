package manifest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSyncConfigBuildsRulesFromRepositories(t *testing.T) {
	temp := t.TempDir()
	const downloadDigest = "blake3:abcdef"
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com/base/",
				Files: []RepositoryFile{
					{
						FileName:       "a.txt",
						OutDir:         "dest",
						Rename:         "renamed.txt",
						DownloadDigest: downloadDigest,
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
	if rule.DownloadChecksum != downloadDigest {
		t.Fatalf("unexpected checksum: %s", rule.DownloadChecksum)
	}
	src := resolved.Sources[rule.Source]
	if src.URL != "https://example.com/base/a.txt" {
		t.Fatalf("unexpected source url: %s", src.URL)
	}
}

func TestResolveSyncConfigCollectsAllRepositoryFiles(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 1,
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

func TestResolveSyncConfigRejectsUnknownEncoding(t *testing.T) {
	temp := t.TempDir()
	cfg := &TaskConfig{
		Version: 1,
		Repositories: []Repository{
			{
				URL: "https://example.com",
				Files: []RepositoryFile{
					{FileName: "a.txt", OutDir: ".", Encoding: "zip"},
				},
			},
		},
	}

	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected invalid encoding error")
	}
}

func TestValidateSyncConfigRejectsMissingSourceURL(t *testing.T) {
	cfg := &SyncConfig{
		Version: "v1",
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
		Version: 1,
		Tasks: map[string]TaskDef{
			"broken": {},
		},
	}
	if _, err := ResolveSyncConfig(cfg, filepath.Join(temp, "vorbere.yaml")); err == nil {
		t.Fatalf("expected validation error for task without run and depends_on")
	}
}

func TestLoadTaskConfigFromRemoteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("version: 1\n"))
	}))
	defer server.Close()

	cfg, err := LoadTaskConfig(server.URL)
	if err != nil {
		t.Fatalf("LoadTaskConfig returned error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version=1, got=%d", cfg.Version)
	}
}

func TestLoadTaskConfigFromRemoteURLReturnsErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := LoadTaskConfig(server.URL); err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}

func TestLoadTaskConfigRejectsUnknownFields(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
repositories:
  - url: https://example.com
    files:
      - file_name: a.txt
        out_dir: .
        artifact_digest: blake3:deadbeef
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadTaskConfig(configPath); err == nil {
		t.Fatalf("expected unknown field error")
	}
}

func TestLoadTaskConfigRejectsLegacyDigestField(t *testing.T) {
	temp := t.TempDir()
	configPath := filepath.Join(temp, "vorbere.yaml")
	content := `version: 1
repositories:
  - url: https://example.com
    files:
      - file_name: a.txt
        out_dir: .
        digest: blake3:deadbeef
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadTaskConfig(configPath); err == nil {
		t.Fatalf("expected unknown field error for legacy digest")
	}
}

func TestIsRemoteConfigLocation(t *testing.T) {
	if !IsRemoteConfigLocation("https://example.com/vorbere.yaml") {
		t.Fatalf("expected https URL to be remote config")
	}
	if !IsRemoteConfigLocation("http://example.com/vorbere.yaml") {
		t.Fatalf("expected http URL to be remote config")
	}
	if IsRemoteConfigLocation("vorbere.yaml") {
		t.Fatalf("expected local path to not be remote config")
	}
}
