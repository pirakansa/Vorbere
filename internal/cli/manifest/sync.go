package manifest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

const (
	outcomeCreated   = "created"
	outcomeUpdated   = "updated"
	outcomeUnchanged = "unchanged"
)

// SyncOptions controls sync behavior.
type SyncOptions struct {
	RootDir   string
	Overwrite bool
	DryRun    bool
	Now       func() time.Time
}

// SyncResult describes sync outcome.
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
}

func Sync(cfg *SyncConfig, opts SyncOptions) (*SyncResult, error) {
	if opts.RootDir == "" {
		return nil, errors.New("root dir is required")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if err := ValidateSyncConfig(cfg); err != nil {
		return nil, err
	}

	rules := cfg.Files

	res := &SyncResult{}
	for _, rule := range rules {
		src := cfg.Sources[rule.Source]
		incoming, err := download(src)
		if err != nil {
			return nil, err
		}
		if err := verifyChecksum(incoming, rule.Checksum); err != nil {
			return nil, err
		}

		target := resolveTargetPath(opts.RootDir, rule.Path)
		outcome, err := applyRule(target, incoming, rule.Mode, opts)
		if err != nil {
			return nil, err
		}

		recordOutcome(res, outcome)
	}
	return res, nil
}

func applyRule(targetPath string, incoming []byte, fileMode string, opts SyncOptions) (string, error) {
	current, err := os.ReadFile(targetPath)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	incomingHash := shared.SHA256Hex(incoming)
	if exists && shared.SHA256Hex(current) == incomingHash {
		return outcomeUnchanged, nil
	}

	backup := shared.BackupTimestamp
	if opts.Overwrite {
		backup = shared.BackupNone
	}

	return writeTarget(targetPath, incoming, current, exists, fileMode, backup, opts)
}

func writeTarget(path string, incoming, current []byte, existed bool, fileMode, backup string, opts SyncOptions) (string, error) {
	if opts.DryRun {
		if existed {
			return outcomeUpdated, nil
		}
		return outcomeCreated, nil
	}
	if existed {
		if err := shared.BackupFile(path, current, backup, opts.Now()); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	perm, err := resolveOutputMode(fileMode)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, incoming, perm); err != nil {
		return "", err
	}
	if existed {
		return outcomeUpdated, nil
	}
	return outcomeCreated, nil
}

func resolveOutputMode(v string) (os.FileMode, error) {
	if strings.TrimSpace(v) == "" {
		return 0o644, nil
	}
	parsed, err := strconv.ParseUint(v, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode %q", v)
	}
	return os.FileMode(parsed), nil
}

func download(src Source) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range src.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed: %s status=%d", src.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func verifyChecksum(content []byte, checksum string) error {
	if checksum == "" {
		return nil
	}
	value := strings.TrimSpace(strings.ToLower(checksum))
	if value == "" {
		return nil
	}
	if shared.BLAKE3Hex(content) != value {
		return errors.New("checksum mismatch")
	}
	return nil
}

func resolveTargetPath(rootDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rootDir, path)
}

func recordOutcome(res *SyncResult, outcome string) {
	switch outcome {
	case outcomeCreated:
		res.Created++
	case outcomeUpdated:
		res.Updated++
	case outcomeUnchanged:
		res.Unchanged++
	}
}
