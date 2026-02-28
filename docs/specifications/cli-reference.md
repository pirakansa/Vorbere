# CLI Reference

## Binary name

`vorbere`

## Global flags

- `--config <path|url>`: path or `http(s)` URL to task config file (default: `vorbere.yaml`)

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

- backup strategy: `timestamp`

Flags:

- `--overwrite`: overwrite existing files without creating timestamp backups
- `--dry-run`: print summary without writing files

### `vorbere plan`

Preview sync operations.

Behavior:

- Equivalent to `vorbere sync --dry-run`.
- Supports `--overwrite`.

### `vorbere completion [bash|zsh|fish|powershell]`

Generate shell completion scripts.

Behavior:

- Prints the completion script to stdout.
- Supports shell-specific subcommands: `bash`, `zsh`, `fish`, `powershell`.
- For shell-specific setup instructions, run `vorbere completion <shell> --help`.

Examples:

- Load bash completion in current session: `source <(vorbere completion bash)`
- Write zsh completion file: `vorbere completion zsh > "${fpath[1]}/_vorbere"`

## Exit codes

- `0`: success
- `2`: configuration/load error
- `4`: undefined task
- `5`: task execution failed
- `6`: sync execution failed
- `1`: other unclassified errors
