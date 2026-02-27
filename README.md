# Vorbere

`vorbere` is a single-binary tool for:

- manifest-driven file sync
- language-agnostic task execution (`run test`, `run lint`, `run ci`)

Manifest format is `version: 3` with `repositories[].files[]` (ppkgmgr-compatible shape).

## Install

```bash
go install github.com/pirakansa/vorbere/cmd/vorbere@latest
```

## Quick Start

```bash
vorbere init
vorbere sync
vorbere run ci
```

## Shell Completion

```bash
vorbere completion --help
```

## Documentation

- User guide: [docs/user-guides/bootkit-devcontainer-workflow.md](docs/user-guides/bootkit-devcontainer-workflow.md)
- CLI reference: [docs/specifications/cli-reference.md](docs/specifications/cli-reference.md)
- Manifest reference: [docs/specifications/manifest-reference.md](docs/specifications/manifest-reference.md)
- Practical examples: [docs/examples/bootkit-devcontainer/](docs/examples/bootkit-devcontainer/)
