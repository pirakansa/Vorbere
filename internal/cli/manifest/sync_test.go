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

func TestSyncDefaultOverwriteAndBackupOverride(t *testing.T) {
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

	opts := SyncOptions{RootDir: temp, LockPath: filepath.Join(temp, LockFileName), Now: func() time.Time { return time.Unix(0, 0) }}
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

	res, err = Sync(cfg, opts)
	if err != nil {
		t.Fatalf("expected default overwrite behavior, got err=%v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}

	content = "v3"
	if err := os.WriteFile(absTarget, []byte("local-change-2"), 0o644); err != nil {
		t.Fatalf("write second local change: %v", err)
	}

	overwriteOpts := opts
	overwriteOpts.ModeOverride = MergeOverwrite
	overwriteOpts.Backup = "timestamp"
	overwriteOpts.Now = func() time.Time { return time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC) }
	res, err = Sync(cfg, overwriteOpts)
	if err != nil {
		t.Fatalf("overwrite sync failed: %v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("expected updated=1 got %+v", res)
	}

	b, err := os.ReadFile(absTarget)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(b) != "v3" {
		t.Fatalf("expected target=v3 got %q", string(b))
	}

	backupPath := fmt.Sprintf("%s.%s.bak", absTarget, "20260227120000")
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file %s: %v", backupPath, err)
	}
}

func TestSyncKeepLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote"))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := filepath.Join(temp, "local.txt")
	if err := os.WriteFile(target, []byte("local"), 0o644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "local.txt", Merge: MergeKeepLocal}},
	}
	res, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: filepath.Join(temp, LockFileName)})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Skipped != 1 {
		t.Fatalf("expected skipped=1 got %+v", res)
	}
}

func TestSyncDryRunDoesNotWriteFileOrLock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("remote"))
	}))
	defer server.Close()

	temp := t.TempDir()
	lockPath := filepath.Join(temp, LockFileName)
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "dryrun.txt"}},
	}

	res, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: lockPath, DryRun: true})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("expected created=1 got %+v", res)
	}
	if _, err := os.Stat(filepath.Join(temp, "dryrun.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no file write during dry-run, err=%v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected no lock write during dry-run, err=%v", err)
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

	if _, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: filepath.Join(temp, LockFileName)}); err != nil {
		t.Fatalf("sync with valid checksum failed: %v", err)
	}

	cfg.Files[0].Path = "checksum-mismatch.txt"
	cfg.Files[0].Checksum = shared.BLAKE3Hex([]byte("different"))
	if _, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: filepath.Join(temp, LockFileName)}); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
}

func TestSyncWritesLockMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("meta"))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := filepath.Join(temp, "lock.txt")
	lockPath := filepath.Join(temp, LockFileName)
	now := time.Date(2026, 2, 27, 15, 0, 0, 0, time.UTC)

	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "lock.txt"}},
	}

	_, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: lockPath, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	lock, err := LoadLock(lockPath)
	if err != nil {
		t.Fatalf("LoadLock failed: %v", err)
	}
	entry, ok := lock.Files[target]
	if !ok {
		t.Fatalf("expected lock entry for %s", target)
	}
	if entry.SourceURL != server.URL {
		t.Fatalf("unexpected source_url: %q", entry.SourceURL)
	}
	expectedHash := shared.SHA256Hex([]byte("meta"))
	if entry.AppliedHash != expectedHash || entry.SourceHash != expectedHash {
		t.Fatalf("unexpected hash entry: %#v", entry)
	}
	if entry.UpdatedAt != now.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", entry.UpdatedAt)
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
	if _, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: filepath.Join(temp, LockFileName)}); err == nil {
		t.Fatalf("expected HTTP status failure")
	}
}

func TestSyncReturnsErrorForCorruptLockFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	temp := t.TempDir()
	lockPath := filepath.Join(temp, LockFileName)
	if err := os.WriteFile(lockPath, []byte(":\n:"), 0o644); err != nil {
		t.Fatalf("write corrupt lock file: %v", err)
	}
	cfg := &SyncConfig{
		Version: "v3",
		Sources: map[string]Source{"src": {URL: server.URL}},
		Files:   []FileRule{{Source: "src", Path: "target.txt"}},
	}
	if _, err := Sync(cfg, SyncOptions{RootDir: temp, LockPath: lockPath}); err == nil {
		t.Fatalf("expected lock parse error")
	}
}
