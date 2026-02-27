# Manifest Reference

This document describes `task.yaml`, `sync.yaml`, and `vorbere.lock` as implemented in the current codebase.

## task.yaml

```yaml
version: v1
tasks:
  test:
    run: "npm test"
    desc: run tests
    env:
      FOO: bar
    cwd: subdir
    depends_on: [lint]
sync:
  ref: sync.yaml
  inline: {}
```

Fields:

- `version`: optional, defaults to `v1` if omitted
- `tasks`: map of task definitions
- `tasks.<name>.run`: shell command (optional when `depends_on` exists)
- `tasks.<name>.desc`: description shown by `tasks list`
- `tasks.<name>.env`: additional environment variables
- `tasks.<name>.cwd`: working directory (absolute or relative to config directory)
- `tasks.<name>.depends_on`: dependency task names
- `sync.ref`: path or HTTP(S) URL to sync manifest
- `sync.inline`: inline sync manifest

Resolution order for sync config:

1. `sync.inline` (highest priority)
2. `sync.ref`
3. `sync.yaml` in the same directory as `task.yaml`
4. empty sync config

## sync.yaml

```yaml
version: v1
sources:
  mysource:
    type: http
    url: https://example.com/file
    headers:
      Authorization: Bearer TOKEN
files:
  - source: mysource
    path: .devcontainer/devcontainer.json
    mode: "0644"
    merge: three_way
    backup: timestamp
    checksum: sha256:<hex>
profiles:
  devcontainer:
    files:
      - source: mysource
        path: /workspaces/.persist/example.txt
        merge: keep_local
```

Fields:

- `version`: optional, defaults to `v1`
- `sources.<id>.type`: currently only `http` is supported
- `sources.<id>.url`: required source URL
- `sources.<id>.headers`: optional HTTP headers
- `files[]`: base sync rules
- `profiles.<name>.files[]`: additional rules appended when `--profile <name>` is used

File rule fields:

- `source`: source ID from `sources`
- `path`: destination path (absolute or relative to config directory)
- `mode`: currently parsed but not applied in writing logic
- `merge`: `three_way` (default), `overwrite`, or `keep_local`
- `backup`: `none` (default) or `timestamp`
- `checksum`: optional `sha256:<hex>` integrity check

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
