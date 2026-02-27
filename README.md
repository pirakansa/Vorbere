# Vorbere

`vorbere` is a single-binary tool that combines:

- Manifest-driven file sync (`sync.yaml` / `task.yaml`)
- Language-agnostic task execution (`run test`, `run lint`, etc.)

## Quick Start

1. Initialize templates:

```bash
vorbere init
```

2. Edit `task.yaml` and `sync.yaml` for your repository.

3. Run sync and tasks:

```bash
vorbere sync
vorbere run test
vorbere run ci
```

## Commands

- `vorbere sync [--mode three_way|overwrite|keep_local] [--backup none|timestamp] [--dry-run]`
- `vorbere run <task> [-- args...]`
- `vorbere tasks list`
- `vorbere plan`
- `vorbere init [--with-sync-ref <path-or-url>]`
