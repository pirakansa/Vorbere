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
	pkgmanifest "github.com/pirakansa/vorbere/pkg/manifest"
)

const (
	MergeThreeWay  = "three_way"
	MergeOverwrite = pkgmanifest.MergeOverwrite
	MergeKeepLocal = "keep_local"

	outcomeCreated   = "created"
	outcomeUpdated   = "updated"
	outcomeUnchanged = "unchanged"
	outcomeSkipped   = "skipped"
)

var ErrSyncConflict = errors.New("sync conflict")

// SyncOptions controls sync behavior.
type SyncOptions struct {
	RootDir      string
	LockPath     string
	ModeOverride string
	Backup       string
	DryRun       bool
	Now          func() time.Time
}

// SyncResult describes sync outcome.
type SyncResult struct {
	Created   int
	Updated   int
	Unchanged int
	Skipped   int
	Conflicts []string
}

func Sync(cfg *SyncConfig, opts SyncOptions) (*SyncResult, error) {
	if opts.RootDir == "" {
		return nil, errors.New("root dir is required")
	}
	if opts.LockPath == "" {
		opts.LockPath = filepath.Join(opts.RootDir, LockFileName)
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if err := ValidateSyncConfig(cfg); err != nil {
		return nil, err
	}

	rules := cfg.Files

	lock, err := LoadLock(opts.LockPath)
	if err != nil {
		return nil, err
	}

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

		mode := chooseMergeMode(rule.Merge, opts.ModeOverride)
		backup := chooseBackup(rule.Backup, opts.Backup)
		outcome, applied, err := applyRule(target, incoming, rule.Mode, mode, backup, lock.Files[target], opts)
		if err != nil {
			if errors.Is(err, ErrSyncConflict) {
				res.Conflicts = append(res.Conflicts, target)
				continue
			}
			return nil, err
		}

		recordOutcome(res, outcome)
		if applied {
			lock.Files[target] = newLockEntry(src.URL, incoming, opts.Now)
		}
	}

	if len(res.Conflicts) > 0 {
		return res, ErrSyncConflict
	}
	if !opts.DryRun {
		if err := SaveLock(opts.LockPath, lock); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func applyRule(targetPath string, incoming []byte, fileMode, mode, backup string, lockEntry LockEntry, opts SyncOptions) (string, bool, error) {
	current, err := os.ReadFile(targetPath)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}

	incomingHash := shared.SHA256Hex(incoming)
	if exists && shared.SHA256Hex(current) == incomingHash {
		return outcomeUnchanged, false, nil
	}

	switch mode {
	case MergeOverwrite:
		return writeTarget(targetPath, incoming, current, exists, fileMode, backup, opts)
	case MergeKeepLocal:
		if exists {
			return outcomeSkipped, false, nil
		}
		return writeTarget(targetPath, incoming, nil, false, fileMode, backup, opts)
	case MergeThreeWay:
		if !exists {
			return writeTarget(targetPath, incoming, nil, false, fileMode, backup, opts)
		}
		appliedHash := lockEntry.AppliedHash
		if appliedHash == "" {
			if shared.SHA256Hex(current) == incomingHash {
				return "unchanged", false, nil
			}
			return "", false, fmt.Errorf("%w: %s has local content but no lock state", ErrSyncConflict, targetPath)
		}
		currentHash := shared.SHA256Hex(current)
		if currentHash == appliedHash {
			return writeTarget(targetPath, incoming, current, true, fileMode, backup, opts)
		}
		if incomingHash == appliedHash {
			return outcomeSkipped, false, nil
		}
		return "", false, fmt.Errorf("%w: %s local and source changed", ErrSyncConflict, targetPath)
	default:
		return "", false, fmt.Errorf("unsupported merge mode: %s", mode)
	}
}

func writeTarget(path string, incoming, current []byte, existed bool, fileMode, backup string, opts SyncOptions) (string, bool, error) {
	if opts.DryRun {
		if existed {
			return outcomeUpdated, false, nil
		}
		return outcomeCreated, false, nil
	}
	if existed {
		if err := shared.BackupFile(path, current, backup, opts.Now()); err != nil {
			return "", false, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	perm, err := resolveOutputMode(fileMode)
	if err != nil {
		return "", false, err
	}
	if err := os.WriteFile(path, incoming, perm); err != nil {
		return "", false, err
	}
	if existed {
		return outcomeUpdated, true, nil
	}
	return outcomeCreated, true, nil
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

func chooseMergeMode(rule, override string) string {
	if override != "" {
		return override
	}
	if rule == "" {
		return MergeOverwrite
	}
	return rule
}

func chooseBackup(rule, override string) string {
	if override != "" {
		return override
	}
	if rule == "" {
		return shared.BackupNone
	}
	return rule
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
	case outcomeSkipped:
		res.Skipped++
	}
}

func newLockEntry(sourceURL string, incoming []byte, now func() time.Time) LockEntry {
	hash := shared.SHA256Hex(incoming)
	return LockEntry{
		SourceURL:   sourceURL,
		AppliedHash: hash,
		SourceHash:  hash,
		UpdatedAt:   now().UTC().Format(time.RFC3339),
	}
}
