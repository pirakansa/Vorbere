package manifest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSyncThreeWayConflictAndOverwrite(t *testing.T) {
	content := "v1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	temp := t.TempDir()
	target := "managed/file.txt"

	cfg := &SyncConfig{
		Version: "v1",
		Sources: map[string]Source{
			"src": {Type: "http", URL: server.URL},
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

	_, err = Sync(cfg, opts)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if err != ErrSyncConflict {
		t.Fatalf("expected ErrSyncConflict got %v", err)
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
	if string(b) != "v2" {
		t.Fatalf("expected target=v2 got %q", string(b))
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
		Version: "v1",
		Sources: map[string]Source{"src": {Type: "http", URL: server.URL}},
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
