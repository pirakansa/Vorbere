package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

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

func resolveOutputMode(value string) (os.FileMode, error) {
	if strings.TrimSpace(value) == "" {
		return 0o644, nil
	}
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file mode %q", value)
	}
	return os.FileMode(parsed), nil
}
