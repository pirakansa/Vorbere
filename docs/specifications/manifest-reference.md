# Manifest Reference

This document defines the `vorbere.yaml` manifest behavior.

Current format: `version: 1`.

Status:

- Implemented behavior: sections without explicit draft markers.

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
        download_digest: <optional <algorithm>:<hex>>
        output_digest: <optional <algorithm>:<hex>>
        rename: AGENTS.md
        mode: "0644"
```

## Top-level fields

- `version`: optional, defaults to `1`
- `vars`: optional string map used by template expansion (`{{ .vars.NAME }}`), where key names must match `[A-Za-z_][A-Za-z0-9_]*`
- `tasks`: map of task definitions
- `repositories`: list of remote repositories to fetch artifacts from

## `tasks` fields

- `tasks.<name>.run`: shell command (optional when `depends_on` exists)
- `tasks.<name>.desc`: description shown by `tasks list`
- `tasks.<name>.env`: additional environment variables
- `tasks.<name>.cwd`: working directory (absolute or relative to config directory)
- `tasks.<name>.depends_on`: dependency task names

## Task Vars and Template Expansion

Goal:

- Centralize version and repeated string management in one place.
- Reuse those values in both `tasks` and `repositories` fields.

Current scope:

- Add top-level `vars` map in `vorbere.yaml`.
- Support string interpolation in selected string fields by using `{{ .vars.NAME }}`.
- Fail fast when referenced vars are undefined.

Non-goals (current):

- No loop/matrix execution yet.
- No expression language beyond direct var lookup.
- No automatic type coercion for non-string fields.

### Schema

```yaml
version: 1

vars:
  GO_VERSION: "1.24.2"
  TOOL_VERSION: "0.3.0"
```

### Vars key constraints

- `vars` keys must match `[A-Za-z_][A-Za-z0-9_]*`.
- Keys outside this pattern are invalid for `{{ .vars.KEY }}` references.

### Expansion targets

Template expansion is applied to string values in:

- `tasks.<name>.run`
- `tasks.<name>.cwd`
- `tasks.<name>.env.<key>`
- `repositories[].url`
- `repositories[].files[].file_name`
- `repositories[].files[].out_dir`
- `repositories[].files[].rename`

### Processing order

1. Load and validate YAML.
2. Resolve template expressions with `vars`.
3. Apply existing environment-variable expansion rules (`$ENV` / `${ENV}` where currently supported).
4. Execute existing task/sync logic unchanged.

### Error behavior

- If a template references an undefined var, config loading fails with exit code `2`.
- Error messages include the field path (for example `tasks.build.run`) and the unresolved key.

### Version management example

```yaml
version: 1

vars:
  GO_VERSION: "1.24.2"
  NODE_VERSION: "24.13.1"

tasks:
  setup-go:
    run: "go install golang.org/dl/go{{ .vars.GO_VERSION }}@latest && go{{ .vars.GO_VERSION }} download"
  print-versions:
    run: "echo go={{ .vars.GO_VERSION }} node={{ .vars.NODE_VERSION }}"

repositories:
  - url: "https://example.com/dist/{{ .vars.NODE_VERSION }}/"
    files:
      - file_name: "node-v{{ .vars.NODE_VERSION }}-linux-x64.tar.xz"
        encoding: tar+xz
        out_dir: "$HOME/.local/lib"
```

Result:

- Version strings are defined once under `vars`.
- Changing `GO_VERSION` or `NODE_VERSION` updates all referenced tasks and artifact paths.

## `repositories` fields

- `repositories[].url`: required base URL
- `repositories[].headers`: optional HTTP headers applied to all files in the repository (`${VAR}` is expanded from environment variables)
- `repositories[].files[]`: file definitions

Supported `repositories[].files[]` fields:

- `file_name` (required): path/name appended to repository base URL
- `out_dir` (required): destination directory (`$ENV` variables are expanded)
- `rename` (optional): output filename override
- `mode` (optional): octal output file mode string (example: `"0755"`)
- `download_digest` (optional): checksum of downloaded artifact in `<algorithm>:<hex>` format
- `output_digest` (optional): checksum of decoded/extracted single output in `<algorithm>:<hex>` format
- `encoding` (optional): `zstd` | `tar+gzip` | `tar+xz`
- `extract` (optional): archive path to extract; omit or `"."` to extract entire archive into `out_dir`

Notes:

- Supported digest algorithms: `blake3`, `sha256`, `md5`.
- `repositories[].headers` expands `${VAR}` placeholders for local config files; undefined variables cause an error.
- When `--config` points to a remote `http(s)` URL, `repositories[].headers` is not expanded and is used as-is.
- Use environment variables for secrets (for example tokens) instead of writing secret values directly in `vorbere.yaml`.
- Header values are masked in error messages.
- `download_digest` is verified before decode/extract.
- `output_digest` is verified only for single-output cases.
- `output_digest` is invalid when extraction resolves to multiple files.
- Legacy fields `digest` and `artifact_digest` are not supported in `version: 1`.
- `rename` and `mode` apply to single-output cases.
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

### zstd decode with both digest checks

```yaml
version: 1

repositories:
  - url: https://example.com/releases/
    files:
      - file_name: tool-linux-amd64.zst
        encoding: zstd
        download_digest: blake3:<hex-of-downloaded-zst>
        output_digest: blake3:<hex-of-decoded-tool>
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
        download_digest: sha256:<hex-of-downloaded-tar.xz>
        out_dir: $HOME/.local/lib
```

### headers with environment variable secrets

```yaml
version: 1

repositories:
  - url: https://example.com/private/
    headers:
      Authorization: "Bearer ${GITHUB_TOKEN}"
    files:
      - file_name: release.txt
        out_dir: .
```

## TODO

- Add per-file OS/architecture selection (for example `os` / `arch` fields under `repositories[].files[]`) so one manifest can switch download targets without maintaining multiple config files.
- Add an explicit opt-in field (for example `allow_header_forward_to`) to permit forwarding repository headers on cross-host redirects only to approved hosts.
- Add task-level precondition and required-variable validation fields (for example `preconditions` / `requires`) so tasks can fail early with clear messages before command execution.
- Add conditional task execution support (for example `if`) to allow skipping commands based on environment or runtime checks.
- Add richer task variable features (for example typed `vars` and loop/matrix execution) to reduce duplicated task definitions.
- Add deferred cleanup support (for example `defer`) so cleanup commands run even when the main task command fails.
- Add `.env` loading support (for example `dotenv` at top-level and task-level) with documented precedence against `env` and process environment variables.
