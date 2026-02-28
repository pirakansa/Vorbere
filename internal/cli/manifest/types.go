package manifest

import pkgmanifest "github.com/pirakansa/vorbere/pkg/manifest"

type TaskConfig = pkgmanifest.TaskConfig
type TaskDef = pkgmanifest.TaskDef
type Repository = pkgmanifest.Repository
type RepositoryFile = pkgmanifest.RepositoryFile
type SymlinkSpec = pkgmanifest.SymlinkSpec
type SyncConfig = pkgmanifest.SyncConfig
type Source = pkgmanifest.Source
type FileRule = pkgmanifest.FileRule

const (
	EncodingZstd    = pkgmanifest.EncodingZstd
	EncodingTarGzip = pkgmanifest.EncodingTarGzip
	EncodingTarXz   = pkgmanifest.EncodingTarXz
)
