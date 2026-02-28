# Bootkit + DevContainer Workflow

This guide shows a practical setup for repositories that want both:

- centralized bootstrap content from `bootkit`
- language-agnostic local tasks through `vorbere run <task>`

## Prerequisites

- `vorbere` is installed in the DevContainer.
- The repository root contains `vorbere.yaml`.
- A persistent mount exists for personal agent state (example: `/workspaces/.persist`).

## 1. Start from the example files

Copy and adapt:

- `docs/examples/bootkit-devcontainer/vorbere.yaml`

Recommended minimum edits:

- Replace task commands with your stack (`npm`, `cargo`, `go`, `make`, etc.).
- Replace repository URLs and file entries with your real paths.
- Update persistent mount target paths for your environment.

## 2. Keep task names stable across repositories

Use the same task names everywhere:

- `sync`
- `fmt`
- `lint`
- `test`
- `build`
- `ci`

Then SKILLS can call only:

```bash
vorbere run <task>
```

without knowing repository-specific tools.

## 3. Sync bootkit-managed files

Preview changes first:

```bash
vorbere sync --dry-run
```

Apply sync (timestamp backup is default):

```bash
vorbere sync
```

Notes:

- Existing files are backed up as timestamped `.bak` files before replacement.
- Use `--overwrite` to skip backup creation.

## 4. Run common tasks

Examples:

```bash
vorbere run fmt
vorbere run lint
vorbere run test
vorbere run ci
```

The actual commands are resolved from `vorbere.yaml`.

## 5. Pattern for personal/auth files in DevContainer

For user-specific files, place them under a persistent mount path and set restrictive `mode` values.

Example pattern:

- `out_dir: /workspaces/.persist/codex`
- `mode: "0600"`

## 6. Operational recommendations

- Use the default overwrite flow for managed bootstrap files.
- Use default sync to keep rollback backups (`.bak`) for large updates.
- Use `--overwrite` when you intentionally want direct replacement.
- Run `vorbere tasks list` after updating `vorbere.yaml`.
