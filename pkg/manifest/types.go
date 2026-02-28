package manifest

// TaskConfig is the repository-level task configuration in vorbere.yaml.
type TaskConfig struct {
	Version      int                `yaml:"version"`
	Tasks        map[string]TaskDef `yaml:"tasks"`
	Repositories []Repository       `yaml:"repositories"`
}

// TaskDef defines one runnable task.
type TaskDef struct {
	Run       string            `yaml:"run"`
	Desc      string            `yaml:"desc"`
	Env       map[string]string `yaml:"env"`
	CWD       string            `yaml:"cwd"`
	DependsOn []string          `yaml:"depends_on"`
}

// Repository groups downloadable file entries under one base URL.
type Repository struct {
	Comment string            `yaml:"_comment"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Files   []RepositoryFile  `yaml:"files"`
}

// RepositoryFile defines one fetch-and-place operation.
type RepositoryFile struct {
	FileName       string       `yaml:"file_name"`
	DownloadDigest string       `yaml:"download_digest"`
	OutputDigest   string       `yaml:"output_digest"`
	Encoding       string       `yaml:"encoding"`
	Extract        string       `yaml:"extract"`
	OutDir         string       `yaml:"out_dir"`
	Rename         string       `yaml:"rename"`
	Mode           string       `yaml:"mode"`
	Symlink        *SymlinkSpec `yaml:"symlink"`
}

// SymlinkSpec is kept for schema compatibility; currently unsupported.
type SymlinkSpec struct {
	Link   string `yaml:"link"`
	Target string `yaml:"target"`
}

// SyncConfig is the normalized internal sync manifest.
type SyncConfig struct {
	Version string            `yaml:"version"`
	Sources map[string]Source `yaml:"sources"`
	Files   []FileRule        `yaml:"files"`
}

// Source defines downloadable resource metadata.
type Source struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

// FileRule defines one target placement operation.
type FileRule struct {
	Source           string `yaml:"source"`
	Path             string `yaml:"path"`
	Mode             string `yaml:"mode"`
	DownloadChecksum string `yaml:"download_checksum"`
	OutputChecksum   string `yaml:"output_checksum"`
	Encoding         string `yaml:"encoding"`
	Extract          string `yaml:"extract"`
	ExpandArchive    bool   `yaml:"expand_archive"`
}
