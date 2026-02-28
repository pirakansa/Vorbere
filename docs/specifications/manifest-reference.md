# Manifest Reference

This document defines the `vorbere.yaml` manifest behavior.

Current format: `version: 1`.

## vorbere.yaml

```yaml
version: 1

tasks:
  test:
    run: "npm test"
    desc: run tests
    env:
      FOO: bar
    cwd: subdir
    depends_on: [lint]

repositories:
  - _comment: bootkit files
    url: https://raw.githubusercontent.com/example/repo/main/
    files:
      - file_name: AGENTS.md
        out_dir: .
        digest: <optional <algorithm>:<hex>>
        rename: AGENTS.md
        mode: "0644"
```

## Top-level fields

- `version`: optional, defaults to `1`
- `tasks`: map of task definitions
- `repositories`: list of remote repositories to fetch artifacts from

## `tasks` fields

- `tasks.<name>.run`: shell command (optional when `depends_on` exists)
- `tasks.<name>.desc`: description shown by `tasks list`
- `tasks.<name>.env`: additional environment variables
- `tasks.<name>.cwd`: working directory (absolute or relative to config directory)
- `tasks.<name>.depends_on`: dependency task names

## `repositories` fields

- `repositories[].url`: required base URL
- `repositories[].headers`: optional HTTP headers applied to all files in the repository
- `repositories[].files[]`: file definitions

Supported `repositories[].files[]` fields:

- `file_name` (required): path/name appended to repository base URL
- `out_dir` (required): destination directory (`$ENV` variables are expanded)
- `rename` (optional): output filename override
- `mode` (optional): octal output file mode string (example: `"0755"`)
- `digest` (optional): checksum of downloaded artifact in `<algorithm>:<hex>` format
- `encoding` (optional): `zstd` | `tar+gzip` | `tar+xz`
- `extract` (optional): archive path to extract; omit or `"."` to extract entire archive into `out_dir`

Notes:

- `digest` validates the downloaded artifact bytes before decode/extract.
- Supported digest algorithms: `blake3`, `sha256`, `md5`.
- `artifact_digest` is removed in `version: 1` and treated as an unknown field.
- `rename` and `mode` apply only to single-output results.
- For multi-output extraction, `mode` is ignored.
- `symlink` remains unsupported.

## `extract` behavior

- `extract` omitted or `"."`: extract entire archive contents into `out_dir`.
- `extract` points to a file entry: produces a single output.
- `extract` points to a directory prefix: extracts all matching children as multiple outputs.

## Backup behavior

Default behavior keeps a timestamp backup before replacing existing files.

`--overwrite` disables backup creation and overwrites directly.

When backup is active, existing destination is copied before overwrite:

`<path>.<YYYYMMDDHHMMSS>.bak`

## Examples

### zstd decode with artifact digest verification

```yaml
version: 1

repositories:
  - url: https://example.com/releases/
    files:
      - file_name: tool-linux-amd64.zst
        encoding: zstd
        digest: blake3:<hex-of-downloaded-zst>
        out_dir: $HOME/.local/bin
        rename: tool
        mode: "0755"
```

### tar.xz full extraction

```yaml
version: 1

repositories:
  - url: https://example.com/dist/
    files:
      - file_name: node-v24.13.1-linux-x64.tar.xz
        encoding: tar+xz
        digest: sha256:<hex-of-downloaded-tar.xz>
        out_dir: $HOME/.local/lib
```
