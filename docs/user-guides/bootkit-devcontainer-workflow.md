# Bootkit + DevContainer Workflow

This guide shows a practical setup for repositories that want both:

- centralized bootstrap content from `bootkit`
- language-agnostic local tasks through `vorbere run <task>`

## Prerequisites

- `vorbere` is installed in the DevContainer.
- The repository root contains `vorbere.yaml` and `sync.yaml`.
- A persistent mount exists for personal agent state (example: `/workspaces/.persist`).

## 1. Start from the example files

Copy and adapt:

- `docs/examples/bootkit-devcontainer/vorbere.yaml`
- `docs/examples/bootkit-devcontainer/sync.yaml`

Recommended minimum edits:

- Replace task commands with your stack (`npm`, `cargo`, `go`, `make`, etc.).
- Replace bootkit URLs with your real paths.
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
vorbere plan --profile devcontainer
```

Apply sync with backup:

```bash
vorbere sync --profile devcontainer --backup timestamp
```

Notes:

- `three_way` detects conflicts when both local and remote changed.
- `overwrite` always writes incoming content.
- `keep_local` keeps existing local files and skips replacement.

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

For user-specific files, put entries in a profile (for example `devcontainer`) and target the persistent mount path.

Example pattern:

- `merge: keep_local`
- `backup: none`

This keeps local identity/auth files intact after container rebuilds.

## 6. Operational recommendations

- Use `three_way` for repository-managed files committed to Git.
- Use `keep_local` for personal files under persistent mounts.
- Use `--backup timestamp` before large updates.
- Run `vorbere tasks list` after updating `vorbere.yaml`.
