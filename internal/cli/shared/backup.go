package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	BackupNone      = "none"
	BackupTimestamp = "timestamp"
)

func BackupFile(path string, content []byte, strategy string, now time.Time) error {
	if strategy != BackupTimestamp {
		return nil
	}
	ts := now.Format("20060102150405")
	backupPath := fmt.Sprintf("%s.%s.bak", path, ts)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(backupPath, content, 0o644)
}
