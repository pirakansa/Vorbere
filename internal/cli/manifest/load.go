package manifest

import (
	"fmt"
	"io"
	"net/http"
	"os"

	pkgmanifest "github.com/pirakansa/vorbere/pkg/manifest"
	"gopkg.in/yaml.v3"
)

func LoadTaskConfig(path string) (*TaskConfig, error) {
	content, err := readConfig(path)
	if err != nil {
		return nil, err
	}

	var cfg TaskConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	pkgmanifest.NormalizeTaskConfig(&cfg)
	return &cfg, nil
}

func IsRemoteConfigLocation(value string) bool {
	return pkgmanifest.IsRemoteConfigLocation(value)
}

func ResolveSyncConfig(taskCfg *TaskConfig, taskConfigPath string) (*SyncConfig, error) {
	_ = taskConfigPath
	if err := pkgmanifest.ValidateTaskConfig(taskCfg); err != nil {
		return nil, err
	}
	return pkgmanifest.BuildSyncConfig(taskCfg)
}

func ValidateSyncConfig(cfg *SyncConfig) error {
	return pkgmanifest.ValidateSyncConfig(cfg)
}

func readConfig(path string) ([]byte, error) {
	if IsRemoteConfigLocation(path) {
		return readRemoteConfig(path)
	}
	return os.ReadFile(path)
}

func readRemoteConfig(location string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("load config failed: %s status=%d", location, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
