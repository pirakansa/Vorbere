# Vorbere

`vorbere` is a single-binary tool for:

- manifest-driven file sync
- language-agnostic task execution (`run test`, `run lint`, `run ci`)

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

## Documentation

- User guide: `docs/user-guides/bootkit-devcontainer-workflow.md`
- CLI reference: `docs/specifications/cli-reference.md`
- Manifest reference: `docs/specifications/manifest-reference.md`
- Practical examples: `docs/examples/bootkit-devcontainer/`
