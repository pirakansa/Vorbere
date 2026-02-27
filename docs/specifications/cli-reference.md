# CLI Reference

## Binary name

`vorbere`

## Global flags

- `--config <path>`: path to task config file (default: `vorbere.yaml`)

## Commands

### `vorbere init`

Create `vorbere.yaml` template.

Behavior:

- Fails if target files already exist.
- Creates only `vorbere.yaml` (single-file manifest).

### `vorbere tasks list`

List task names from `vorbere.yaml`.

### `vorbere run <task> [-- args...]`

Run one task from `vorbere.yaml`.

Behavior:

- Executes task commands via `bash -lc`.
- Resolves and runs `depends_on` first.
- Fails on undefined task.

### `vorbere sync`

Sync files from `repositories` in `vorbere.yaml`.

Default behavior:

- merge mode: `overwrite`
- backup strategy: `none`

Flags:

- `--mode three_way|overwrite|keep_local`: override merge mode
- `--backup none|timestamp`: override backup strategy
- `--dry-run`: print summary without writing files

### `vorbere plan`

Preview sync operations.

Behavior:

- Equivalent to `vorbere sync --dry-run`.
- Supports `--mode` and `--backup`.

## Exit codes

- `0`: success
- `2`: configuration/load error
- `3`: sync conflict
- `4`: undefined task
- `5`: task execution failed
- `6`: sync execution failed
- `1`: other unclassified errors
