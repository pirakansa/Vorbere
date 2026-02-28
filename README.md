# Vorbere

`vorbere` is a single-binary tool for:

- manifest-driven file sync
- language-agnostic task execution (`run test`, `run lint`, `run ci`)

Manifest format is `version: 1` with `repositories[].files[]`.

## Install

```bash
go install github.com/pirakansa/vorbere/cmd/vorbere@latest
```

Or install a published release binary:

```bash
curl -fsSL https://raw.githubusercontent.com/pirakansa/vorbere/main/install.sh | bash
```

## Quick Start

```bash
vorbere init
vorbere sync
vorbere run ci
```

## GitHub Action

```yaml
- uses: pirakansa/vorbere@main
  with:
    version: latest

- run: vorbere --config vorbere.yaml run lint
- run: vorbere --config vorbere.yaml run test
- run: vorbere --config vorbere.yaml run build
```

For example, replace Makefile-based CI steps with:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '^1.24.0'

- uses: pirakansa/vorbere@main
  with:
    version: latest

- run: vorbere --config vorbere.yaml run lint
- run: vorbere --config vorbere.yaml run test
- run: vorbere --config vorbere.yaml run build
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
- Encoding examples: [docs/examples/manifest-encodings/](docs/examples/manifest-encodings/)
