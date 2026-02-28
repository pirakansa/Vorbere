# Manifest Reference

This document defines the `vorbere.yaml` manifest behavior.

Current format: `version: 3` (ppkgmgr-compatible repository/file shape).

## vorbere.yaml

```yaml
version: 3

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
    url: https://raw.githubusercontent.com/pirakansa/bootkit/main/
    files:
      - file_name: AGENTS.md
        out_dir: .
        digest: <optional hex>
        rename: AGENTS.md
        mode: "0644"
      - file_name: templates/codex/auth.json
        out_dir: /workspaces/.persist/codex
        rename: auth.json
```

### Top-level fields

- `version`: optional, defaults to `3`
- `tasks`: map of task definitions
- `repositories`: list of remote repositories to fetch artifacts from

### `tasks` fields

- `tasks.<name>.run`: shell command (optional when `depends_on` exists)
- `tasks.<name>.desc`: description shown by `tasks list`
- `tasks.<name>.env`: additional environment variables
- `tasks.<name>.cwd`: working directory (absolute or relative to config directory)
- `tasks.<name>.depends_on`: dependency task names

### `repositories` fields

- `repositories[].url`: required base URL
- `repositories[].headers`: optional HTTP headers applied to all files in the repository
- `repositories[].files[]`: file definitions

Supported `repositories[].files[]` fields:

- `file_name` (required): path/name appended to repository base URL
- `out_dir` (required): destination directory (`$ENV` variables are expanded)
- `rename` (optional): output filename override
- `mode` (optional): octal output file mode string (example: `"0755"`)
- `digest` (optional): BLAKE3 hex digest of final output (post-decode/extract)
- `artifact_digest` (optional): BLAKE3 hex digest of the downloaded artifact before decode/extract
- `encoding` (optional): `zstd` | `tar+gzip` | `tar+xz`
- `extract` (optional): archive path to extract; omit or `"."` to extract entire archive into `out_dir`

Notes:

- `digest` and `artifact_digest` are plain BLAKE3 hex strings (no `algo:` prefix).
- Processing flow is `artifact_digest` verify -> decode/extract -> `digest` verify.
- For archive full extraction (`extract` omitted or `"."`), `digest` is not supported.
- `rename` and `mode` apply to single-output cases, and are ignored for full-archive extraction.
- `symlink` remains unsupported.

## Backup behavior

Default behavior keeps a timestamp backup before replacing existing files.

`--overwrite` disables backup creation and overwrites directly.

When backup is active, existing destination is copied before overwrite:

`<path>.<YYYYMMDDHHMMSS>.bak`
