<p align="center">
  <img src="./docs/vorbere.jpg" width="320" alt="vorbere logo"/>
</p>

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

For repository auth headers, prefer environment variables in `vorbere.yaml`:

```yaml
repositories:
  - url: https://example.com/private/
    headers:
      Authorization: "Bearer ${GITHUB_TOKEN}"
```

When `--config` uses a remote `http(s)` URL, `repositories[].headers` values are treated as literals (no `${VAR}` expansion).

You can centralize versions with top-level `vars` and reuse them in task and repository fields:

```yaml
vars:
  NODE_VERSION: "24.13.1"

tasks:
  print-node:
    run: "echo ${{ .vars.NODE_VERSION }}"

repositories:
  - url: "https://example.com/dist/${{ .vars.NODE_VERSION }}/"
    files:
      - file_name: "node-v${{ .vars.NODE_VERSION }}-linux-x64.tar.xz"
        out_dir: "$HOME/.local/lib"
```

## GitHub Action

```yaml
- uses: pirakansa/vorbere@v0
  with:
    version: v0.3.0

- run: vorbere --config vorbere.yaml run lint
- run: vorbere --config vorbere.yaml run test
- run: vorbere --config vorbere.yaml run build
```

For example, replace Makefile-based CI steps with:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '^1.24.0'

- uses: pirakansa/vorbere@v0
  with:
    version: v0.3.0

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
