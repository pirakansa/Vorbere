# Manifest Reference

This document defines the `vorbere.yaml` manifest and `vorbere.lock` behavior.

Current format: `version: 3` (ppkgmgr-compatible shape with `vorbere` extensions).

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
        x_vorbere:
          merge: three_way
          backup: timestamp
      - file_name: templates/codex/auth.json
        out_dir: /workspaces/.persist/codex
        rename: auth.json
        x_vorbere:
          profile: devcontainer
          merge: keep_local
          backup: none
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
- `digest` (optional): SHA-256 hex digest of final output
- `x_vorbere.merge` (optional): `three_way` (default), `overwrite`, `keep_local`
- `x_vorbere.backup` (optional): `none` (default), `timestamp`
- `x_vorbere.profile` (optional): include this file only when `--profile <name>` matches

Currently unsupported in `vorbere` (explicitly rejected when set):

- `artifact_digest`
- `encoding`
- `extract`
- `symlink`

## Merge behavior

- `three_way`:
  - create file if missing
  - update when local hash equals last applied hash
  - conflict if local changed and incoming changed
  - skip when incoming equals last applied hash
- `overwrite`:
  - always write incoming content
- `keep_local`:
  - keep existing local file, create only when missing

## Backup behavior

When backup is `timestamp`, existing destination is copied before overwrite:

`<path>.<YYYYMMDDHHMMSS>.bak`

## Lock file (`vorbere.lock`)

Stored at repository root (same root used for command config).

Used for three-way conflict detection and update tracking.

Top-level fields:

- `version`
- `files.<absolute-target-path>.source_url`
- `files.<absolute-target-path>.applied_hash`
- `files.<absolute-target-path>.source_hash`
- `files.<absolute-target-path>.updated_at`
