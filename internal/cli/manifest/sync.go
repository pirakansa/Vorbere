package manifest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

const (
	MergeThreeWay  = "three_way"
	MergeOverwrite = "overwrite"
	MergeKeepLocal = "keep_local"
)

var ErrSyncConflict = errors.New("sync conflict")

// SyncOptions controls sync behavior.
type SyncOptions struct {
	RootDir      string
	LockPath     string
	ModeOverride string
	Backup       string
	DryRun       bool
	Profile      string
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

	rules, err := ResolveProfileFiles(cfg, opts.Profile)
	if err != nil {
		return nil, err
	}

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

		target := rule.Path
		if !filepath.IsAbs(target) {
			target = filepath.Join(opts.RootDir, target)
		}

		mode := chooseMergeMode(rule.Merge, opts.ModeOverride)
		backup := chooseBackup(rule.Backup, opts.Backup)
		outcome, applied, err := applyRule(target, incoming, mode, backup, lock.Files[target], opts)
		if err != nil {
			if errors.Is(err, ErrSyncConflict) {
				res.Conflicts = append(res.Conflicts, target)
				continue
			}
			return nil, err
		}

		switch outcome {
		case "created":
			res.Created++
		case "updated":
			res.Updated++
		case "unchanged":
			res.Unchanged++
		case "skipped":
			res.Skipped++
		}
		if applied {
			lock.Files[target] = LockEntry{
				SourceURL:   src.URL,
				AppliedHash: shared.SHA256Hex(incoming),
				SourceHash:  shared.SHA256Hex(incoming),
				UpdatedAt:   opts.Now().UTC().Format(time.RFC3339),
			}
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

func applyRule(targetPath string, incoming []byte, mode, backup string, lockEntry LockEntry, opts SyncOptions) (string, bool, error) {
	current, err := os.ReadFile(targetPath)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}

	incomingHash := shared.SHA256Hex(incoming)
	if exists && shared.SHA256Hex(current) == incomingHash {
		return "unchanged", false, nil
	}

	switch mode {
	case MergeOverwrite:
		return writeTarget(targetPath, incoming, current, exists, backup, opts)
	case MergeKeepLocal:
		if exists {
			return "skipped", false, nil
		}
		return writeTarget(targetPath, incoming, nil, false, backup, opts)
	case MergeThreeWay:
		if !exists {
			return writeTarget(targetPath, incoming, nil, false, backup, opts)
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
			return writeTarget(targetPath, incoming, current, true, backup, opts)
		}
		if incomingHash == appliedHash {
			return "skipped", false, nil
		}
		return "", false, fmt.Errorf("%w: %s local and source changed", ErrSyncConflict, targetPath)
	default:
		return "", false, fmt.Errorf("unsupported merge mode: %s", mode)
	}
}

func writeTarget(path string, incoming, current []byte, existed bool, backup string, opts SyncOptions) (string, bool, error) {
	if opts.DryRun {
		if existed {
			return "updated", false, nil
		}
		return "created", false, nil
	}
	if existed {
		if err := shared.BackupFile(path, current, backup, opts.Now()); err != nil {
			return "", false, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(path, incoming, 0o644); err != nil {
		return "", false, err
	}
	if existed {
		return "updated", true, nil
	}
	return "created", true, nil
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
		return MergeThreeWay
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
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) != 2 {
		return errors.New("checksum must be <algo>:<value>")
	}
	algo := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])
	if algo != "sha256" {
		return fmt.Errorf("unsupported checksum algo: %s", algo)
	}
	if shared.SHA256Hex(content) != value {
		return errors.New("checksum mismatch")
	}
	return nil
}
