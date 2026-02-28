package manifest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

func TestSyncDefaultMakesTimestampBackupOnUpdate(t *testing.T) {
	content := "v1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := "managed/file.txt"

	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{
			"src": {URL: server.URL},
		},
		Files: []FileRule{{Source: "src", Path: target}},
	}

	opts := SyncOptions{RootDir: temp, Now: func() time.Time { return time.Unix(0, 0) }}
	res, err := Sync(cfg, opts)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("expected created=1 got %+v", res)
	}

	content = "v2"
	absTarget := filepath.Join(temp, target)
	if err := os.WriteFile(absTarget, []byte("local-change"), 0o644); err != nil {
		t.Fatalf("write local change: %v", err)
	}

	updateOpts := opts
	updateOpts.Now = func() time.Time { return time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC) }
	res, err = Sync(cfg, updateOpts)
	if err != nil {
		t.Fatalf("expected default sync success, got err=%v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}

	b, err := os.ReadFile(absTarget)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(b) != "v2" {
		t.Fatalf("expected target=v2 got %q", string(b))
	}

	backupPath := fmt.Sprintf("%s.%s.bak", absTarget, "20260227120000")
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file %s: %v", backupPath, err)
	}
}

func TestSyncOverwriteFlagSkipsTimestampBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote-v2"))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := filepath.Join(temp, "overwrite.txt")
	if err := os.WriteFile(target, []byte("local"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "overwrite.txt"}},
	}
	now := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	res, err := Sync(cfg, SyncOptions{RootDir: temp, Overwrite: true, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}
	backupPath := fmt.Sprintf("%s.%s.bak", target, "20260228100000")
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file with overwrite flag, err=%v", err)
	}
}

func TestSyncDryRunDoesNotWriteFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "dryrun.txt"}},
	}

	res, err := Sync(cfg, SyncOptions{RootDir: temp, DryRun: true})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("expected created=1 got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(temp, "dryrun.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no file write during dry-run, err=%v", err)
	}
}

func TestSyncChecksumValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files: []FileRule{
			{
				Source:   "src",
				Path:     "checksum.txt",
				Checksum: shared.BLAKE3Hex([]byte("content")),
			},
		},
	}

	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err != nil {
		t.Fatalf("sync with valid checksum failed: %v", err)
	}

	cfg.Files[0].Path = "checksum-mismatch.txt"
	cfg.Files[0].Checksum = shared.BLAKE3Hex([]byte("different"))
	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
}

func TestSyncReturnsErrorForHTTPStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "fail.txt"}},
	}
	if _, err := Sync(cfg, SyncOptions{RootDir: temp}); err == nil {
		t.Fatalf("expected HTTP status failure")
	}
}

func TestSyncReportsPerFileProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	temp := t.TempDir()
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files: []FileRule{
			{Source: "src", Path: "a.txt"},
			{Source: "src", Path: "b.txt"},
		},
	}

	var progress []SyncFileProgress
	_, err := Sync(cfg, SyncOptions{
		RootDir: temp,
		OnFile: func(item SyncFileProgress) {
			progress = append(progress, item)
		},
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(progress) != 2 {
		t.Fatalf("expected 2 progress entries, got %d", len(progress))
	}
	if progress[0].Index != 1 || progress[0].Total != 2 || progress[0].Outcome != outcomeCreated {
		t.Fatalf("unexpected first progress entry: %#v", progress[0])
	}
	if progress[1].Index != 2 || progress[1].Total != 2 || progress[1].Outcome != outcomeCreated {
		t.Fatalf("unexpected second progress entry: %#v", progress[1])
	}
}
