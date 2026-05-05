<p align="center">
  <img src="assets/rudder-logo.svg" alt="Rudder" width="820"/>
</p>

<p align="center">
  A high-performance, multi-cluster <code>kubectl</code> wrapper for managing Kubernetes environments with style.
</p>

<p align="center">
  <a href="https://github.com/dcrespo1/rudder/actions/workflows/ci.yml">
  <img src="https://img.shields.io/github/actions/workflow/status/dcrespo1/rudder/ci.yml?branch=main&label=CI" alt="CI"/>
</a>
  <a href="https://github.com/dcrespo1/rudder/releases/latest">
    <img src="https://img.shields.io/github/v/release/dcrespo1/rudder" alt="Latest Release"/>
  </a>
  <a href="https://github.com/dcrespo1/rudder/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/dcrespo1/rudder" alt="License"/>
  </a>
  <a href="https://go.dev/doc/devel/release">
    <img src="https://img.shields.io/badge/go-1.26.1-blue" alt="Go Version"/>
  </a>
</p>

---

## What is Rudder?

Rudder is a `kubectl` wrapper that lets you manage multiple Kubernetes cluster environments from a single, fast CLI. Register your kubeconfigs once, switch between clusters with a fuzzy picker, and run `kubectl` commands against any environment — all without touching `~/.kube/config` directly.

- **Fuzzy environment picker** with live cluster reachability status
- **Adaptive terminal theming** — looks great on any terminal color scheme
- **Fan-out execution** — run a command across all clusters sequentially
- **Zero side effects** — never mutates your kubeconfig files
- **Fast** — direct `kubectl` passthrough with no overhead

---

## Demo

```
❯ rudder use
  ● kind-rudder-dev       Local dev cluster          [dev, test]       ✓ reachable
  ● kind-rudder-staging   Staging environment        [staging, stg]    ✓ reachable
  ● kind-rudder-prod      Production cluster         [prod]            ✓ reachable

> kind-rudder-staging
```

---

## Installation

### go install

```sh
go install github.com/dcrespo1/rudder/cmd/rudder@latest
```

### Direct download

```sh
curl -fsSL https://raw.githubusercontent.com/dcrespo1/rudder/main/scripts/install.sh | sh
```

### Build from source

```sh
git clone https://github.com/dcrespo1/rudder.git
cd rudder
just build
```

Binary will be at `./bin/rudder`.

---

## Quick Start

```sh
# First-time setup — scan for kubeconfigs and register environments
rudder init

# Or add environments manually
rudder config add --name dev \
  --kubeconfig ~/.kube/kind-rudder-dev.yaml \
  --context kind-rudder-dev

# List all environments
rudder envs

# Switch environment (fuzzy picker)
rudder use

# Switch directly
rudder use staging

# Run kubectl against the active environment
rudder exec -- get pods -n kube-system

# One-shot override
RUDDER_ENV=prod rudder exec -- get nodes
```

---

## Commands

### `rudder init`

Interactive first-time setup. Scans for existing kubeconfigs and walks you through registering environments.

### `rudder envs`

List all registered environments with live status indicators.

```sh
rudder envs
rudder envs --tags govcloud
rudder envs --output json
```

### `rudder use [env]`

Switch the active environment. Without an argument, opens a full-screen fuzzy picker with live cluster reachability probed concurrently in the background.

```sh
rudder use              # fuzzy picker
rudder use staging      # direct switch
```

### `rudder exec`

Run any `kubectl` command against the active environment. The correct `KUBECONFIG` and `--context` are injected automatically.

```sh
rudder exec -- get pods -A
rudder exec -- apply -f manifest.yaml
rudder exec -- logs -f deployment/myapp -n myns
```

### `rudder exec --all`

Fan out a command across all environments sequentially, with a styled cluster header per environment.

```sh
rudder exec --all -- get nodes
rudder exec --all --tags staging,prod -- get pods -n platform
```

### `rudder config`

Manage the environment registry.

```sh
rudder config add
rudder config add --name prod --kubeconfig ~/.kube/prod.yaml --context prod-admin
rudder config remove staging
rudder config rename staging aks-staging
rudder config list
rudder config edit        # open config in $EDITOR
rudder config validate    # verify all kubeconfig paths and contexts
```

### `rudder version`

```sh
rudder version
rudder version --short
```

---

## Configuration

Rudder stores its config at `~/.rudder/config.yaml`:

```yaml
version: "1"

kubectl_path: "" # leave empty to resolve from PATH
default_namespace: ""

environments:
  - name: dev
    description: "Local dev cluster"
    kubeconfig: ~/.kube/kind-rudder-dev.yaml
    context: kind-rudder-dev
    namespace: default
    tags: [dev, local]

  - name: staging
    description: "Staging AKS cluster"
    kubeconfig: ~/.kube/aks-staging.yaml
    context: aks-staging-admin
    namespace: platform
    tags: [azure, aks, staging]

  - name: prod
    description: "Production AKS cluster"
    kubeconfig: ~/.kube/aks-prod.yaml
    context: aks-prod-admin
    namespace: platform
    tags: [azure, aks, prod]
```

The active environment is tracked separately in `~/.rudder/state.yaml` and written atomically — safe for concurrent terminal sessions.

---

## Environment Variables

| Variable          | Description                                           |
| ----------------- | ----------------------------------------------------- |
| `RUDDER_CONFIG`   | Override path to config dir (default: `~/.rudder/`)   |
| `RUDDER_ENV`      | Override active environment for a single invocation   |
| `RUDDER_KUBECTL`  | Path to kubectl binary                                |
| `RUDDER_NO_COLOR` | Disable color output                                  |
| `RUDDER_NO_TUI`   | Disable interactive TUI, force plain output           |
| `RUDDER_CI`       | Set to `true` in CI — implies `NO_TUI` and `NO_COLOR` |

---

## Shell Completions

```sh
# Zsh (oh-my-zsh)
mkdir -p ~/.oh-my-zsh/completions
rudder completion zsh > ~/.oh-my-zsh/completions/_rudder
exec zsh

# Bash
echo 'source <(rudder completion bash)' >> ~/.bashrc

# Fish
rudder completion fish > ~/.config/fish/completions/rudder.fish

# Or add to your .zshrc for always up-to-date completions
source <(rudder completion zsh)
```

---

## Development

### Prerequisites

- Go 1.26+
- [`just`](https://github.com/casey/just)
- `kubectl` on PATH
- `golangci-lint` for linting
- `goreleaser` for release builds

### Common tasks

```sh
just build          # build ./bin/rudder
just test           # run tests
just lint           # golangci-lint
just fmt            # gofmt + goimports
just run envs       # run locally
just run use dev    # run locally with args
just completions    # generate shell completions
just release-dry    # goreleaser snapshot build
just check          # fmt + lint + test (pre-commit)
```

---

## Project Structure

```
rudder/
├── cmd/rudder/         # CLI entrypoint and cobra commands
├── internal/
│   ├── config/         # config schema, loading, validation
│   ├── kubeconfig/     # kubeconfig merging and context management
│   ├── kubectl/        # kubectl exec and passthrough
│   ├── cluster/        # lightweight cluster reachability checks
│   └── ui/             # bubbletea TUI, lipgloss theme, output helpers
├── pkg/rudder/         # version constants and build info
├── docs/               # documentation
├── scripts/            # install.sh
└── assets/             # logo and static assets
```

---

## License

MIT — see [LICENSE](LICENSE)
