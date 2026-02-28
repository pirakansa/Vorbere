package manifest

import (
	"errors"
	"path/filepath"
	"time"
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
	OnFile    func(SyncFileProgress)
}

// SyncFileProgress describes one processed file during sync.
type SyncFileProgress struct {
	Index   int
	Total   int
	Path    string
	Outcome string
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
	total := len(rules)
	for index, rule := range rules {
		src := cfg.Sources[rule.Source]
		artifact, err := download(src)
		if err != nil {
			return nil, err
		}
		if err := verifyChecksum(artifact, rule.DownloadChecksum); err != nil {
			return nil, err
		}

		target := resolveTargetPath(opts.RootDir, rule.Path)
		outcome, err := applyProcessedRule(target, artifact, rule, opts)
		if err != nil {
			return nil, err
		}

		recordOutcome(res, outcome)
		if opts.OnFile != nil {
			opts.OnFile(SyncFileProgress{
				Index:   index + 1,
				Total:   total,
				Path:    target,
				Outcome: outcome,
			})
		}
	}
	return res, nil
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
