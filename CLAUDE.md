# CLAUDE.md — Rudder

> Rudder is a high-performance, multi-cluster `kubectl` wrapper written in Go.
> It provides a unified CLI for managing kubeconfigs, switching cluster contexts,
> and proxying `kubectl` commands across multiple Kubernetes environments.
> Rudder is designed to be sleek, fast, and visually polished — a tool you
> enjoy using every day.

---

## Project Overview

| Field        | Value                           |
|--------------|---------------------------------|
| Language     | Go 1.22+                        |
| Binary name  | `rudder`                        |
| Module path  | `gitlab.com/dcresp0/rudder` |
| CLI library  | `cobra` + `viper`               |
| Config store | `~/.rudder/`                    |
| License      | MIT                             |

---

## Repository Layout

```
rudder/
├── cmd/
│   └── rudder/
│       ├── main.go           # Entrypoint
│       ├── root.go           # Root cobra command, global flags, PersistentPreRun
│       ├── init.go           # `rudder init` — first-time setup wizard
│       ├── config.go         # `rudder config` subcommands (add, remove, list, rename)
│       ├── use.go            # `rudder use [env]` — fuzzy picker or direct switch
│       ├── envs.go           # `rudder envs` — list all registered environments
│       ├── exec.go           # `rudder exec` / passthrough to kubectl
│       └── version.go        # `rudder version`
├── internal/
│   ├── config/
│   │   ├── config.go         # Rudder config schema (RudderConfig, Environment)
│   │   ├── loader.go         # Load/save ~/.rudder/config.yaml
│   │   └── validate.go       # Validate kubeconfig paths and contexts
│   ├── kubeconfig/
│   │   ├── merge.go          # Merge multiple kubeconfigs into a temp unified config
│   │   ├── context.go        # Get/set current-context in a kubeconfig
│   │   └── discover.go       # Auto-discover kubeconfigs from known paths
│   ├── kubectl/
│   │   ├── exec.go           # Exec kubectl with KUBECONFIG env injection
│   │   └── resolve.go        # Resolve kubectl binary path (PATH, override)
│   ├── cluster/
│   │   └── ping.go           # Lightweight cluster reachability check (for status indicators)
│   └── ui/
│       ├── theme.go          # Adaptive lip gloss styles (built from terminal bg detection)
│       ├── picker.go         # Fuzzy env picker with live status (bubbletea model)
│       ├── table.go          # Tabular output (bubbles/table)
│       ├── spinner.go        # Spinner wrapper for async ops
│       ├── header.go         # Rudder branded header/banner
│       └── output.go         # Structured stdout/stderr helpers
├── pkg/
│   └── rudder/
│       └── version.go        # Version constants, build info
├── configs/
│   └── config.example.yaml   # Annotated example Rudder config
├── docs/
│   ├── setup.md              # Setup & installation guide
│   ├── usage.md              # Command reference
│   └── environments.md       # Environment & kubeconfig management
├── scripts/
│   └── install.sh            # Curl-pipe installer
├── completions/              # Generated shell completions (gitignored)
├── .goreleaser.yaml          # GoReleaser config for cross-platform builds
├── justfile
├── go.mod
├── go.sum
└── CLAUDE.md
```

---

## Core Concepts

### Environment
A named alias for a Kubernetes cluster context. An environment maps to:
- A path to a kubeconfig file
- A specific context name within that kubeconfig (optional; defaults to `current-context`)
- Optional metadata (description, tags, namespace override)

### Active Environment
The currently selected environment, stored in `~/.rudder/state.yaml`. All passthrough
`kubectl` commands run against the active environment's kubeconfig/context.

### Kubeconfig Merging
Rudder can optionally merge all registered kubeconfigs into a single ephemeral
kubeconfig (written to a temp file) and switch contexts within it — matching
the native `KUBECONFIG=a:b:c` merge behavior but with a named, stable interface.

---

## Rudder Config Schema

`~/.rudder/config.yaml`:

```yaml
version: "1"

# Path to kubectl binary; omit to resolve from PATH
kubectl_path: ""

# Default namespace override applied to all exec commands (can be overridden per-env)
default_namespace: ""

environments:
  - name: dev
    description: "Local dev cluster (kind)"
    kubeconfig: ~/.kube/kind-dev.yaml
    context: kind-dev          # optional; uses current-context if omitted
    namespace: default
    tags: [local, kind]

  - name: staging
    description: "Azure GovCloud staging AKS"
    kubeconfig: ~/.kube/aks-staging.yaml
    context: aks-staging-admin
    namespace: platform
    tags: [azure, aks, govcloud]

  - name: prod
    description: "Production AKS (GCC High)"
    kubeconfig: ~/.kube/aks-prod.yaml
    context: aks-prod-admin
    namespace: platform
    tags: [azure, aks, govcloud, prod]
```

`~/.rudder/state.yaml`:

```yaml
active_environment: staging
```

---

## Command Reference

### `rudder init`
Interactive first-time setup. Scans for existing kubeconfigs, prompts the user
to register environments, and writes `~/.rudder/config.yaml`.

```
rudder init
```

### `rudder envs`
List all registered environments with status indicators. Highlights the active one.

```
rudder envs
rudder envs --tags govcloud
rudder envs --output json
```

### `rudder use [env]`
Switch the active environment. When called with no argument, opens a full-screen
fuzzy picker (bubbletea) that shows all environments with live reachability status
probed concurrently in the background.

```
rudder use              # opens fuzzy picker with live cluster status
rudder use staging      # direct switch, no picker
```

**Fuzzy picker behavior:**
- Environments load instantly from config; status indicators populate async
- Status icons: `●` reachable (green), `●` unreachable (red), `◌` probing (dim)
- Shows environment name, description, tags, and active marker
- Fuzzy search filters by name, description, and tags
- `Enter` to select, `Esc`/`q` to cancel

### `rudder config`
Manage the environment registry.

```
rudder config add                         # interactive wizard
rudder config add --name dev \
  --kubeconfig ~/.kube/dev.yaml \
  --context kind-dev

rudder config remove staging
rudder config rename staging aks-staging
rudder config list
rudder config edit                        # open config in $EDITOR
rudder config validate                    # check all kubeconfig paths/contexts
```

### `rudder exec` / Passthrough
Run any `kubectl` command against the active environment.

```
# Explicit exec subcommand
rudder exec -- get pods -n kube-system

# One-shot env override
RUDDER_ENV=prod rudder exec -- get nodes
```

The active environment's kubeconfig is injected as `KUBECONFIG=<path>` and
`--context=<context>` is prepended automatically.

**No active environment set:**
If `state.yaml` is absent or `active_environment` is empty, Rudder does **not**
fall back to the ambient `KUBECONFIG` env var or `~/.kube/config`. It exits with
a clear error:

```
error: no active environment set — run `rudder use` to select one
```

Silent fallback to ambient kubeconfig would make Rudder's behavior
unpredictable and could cause accidental operations against an unintended cluster.
The `RUDDER_ENV` override is the only escape hatch for one-shot invocations
without a persisted active environment.

### `rudder exec --all`
Fan out a kubectl command across all registered environments sequentially.
Output is grouped under a styled cluster header per environment.

```
rudder exec --all -- get nodes
rudder exec --all --tags govcloud -- get pods -n platform
```

Output format:
```
─── staging ──────────────────────────────────────────
NAME                STATUS   ROLES   AGE
aks-node-001        Ready    agent   12d

─── prod ─────────────────────────────────────────────
NAME                STATUS   ROLES   AGE
aks-node-001        Ready    agent   45d
aks-node-002        Ready    agent   45d
```

### `rudder version`
```
rudder version
rudder version --short
```

---

## UI & Visual Design

### Design Philosophy
- **Adaptive theming**: Rudder detects terminal background luminance and builds
  its lip gloss style palette accordingly — no hardcoded dark/light assumption.
  Users with Solarized, Gruvbox, Tokyo Night, or plain white terminals all get
  a coherent experience.
- **Minimal noise**: Decorative chrome is used sparingly. Output is dense and
  information-rich, not padded with unnecessary borders or banners.
- **Progressive disclosure**: Simple commands produce simple output. Rich TUI
  (spinners, pickers, tables) only activates for interactive use cases.
- **Zero flicker**: All async status probes update in-place via bubbletea's
  model/update/view loop — no terminal scrolljacking.

### Charmbracelet Stack

| Package                          | Role                                             |
|----------------------------------|--------------------------------------------------|
| `charmbracelet/bubbletea`        | TUI framework for interactive commands           |
| `charmbracelet/bubbles`          | Spinner, table, text input, list components      |
| `charmbracelet/lipgloss`         | Adaptive styling, borders, color, layout         |
| `charmbracelet/glamour`          | Markdown rendering for help and doc output       |
| `muesli/termenv`                 | Terminal capability detection, background color  |

### Adaptive Theme Implementation

Theme is constructed once at startup in `internal/ui/theme.go` using `termenv`
to detect the terminal background color, then building a `lipgloss.AdaptiveColor`
palette. All UI components consume the theme via a passed `Theme` struct — no
package-level style globals.

```go
// internal/ui/theme.go
type Theme struct {
    Primary   lipgloss.AdaptiveColor
    Muted     lipgloss.AdaptiveColor
    Success   lipgloss.AdaptiveColor
    Warning   lipgloss.AdaptiveColor
    Danger    lipgloss.AdaptiveColor
    Active    lipgloss.Style
    Header    lipgloss.Style
    Separator lipgloss.Style
    Badge     lipgloss.Style
}

func NewTheme() Theme {
    // Uses lipgloss.AdaptiveColor{Light: "...", Dark: "..."} throughout
    // so colors are appropriate whether the terminal is light or dark
}
```

### TUI Failure Handling

Bubbletea programs can fail at startup (terminal too narrow, non-TTY pipe, `TERM`
unset) or mid-render (panic in `View`, write error on stdout). Rudder handles both:

- **Non-TTY / pipe detected**: before starting any bubbletea program, check
  `term.IsTerminal(int(os.Stdout.Fd()))`. If false, skip the TUI entirely and
  fall back to `PlainView()` output. This covers `rudder use` piped into a script.
- **Terminal too small**: the picker's `Update` func handles `tea.WindowSizeMsg`
  and renders a `too small — resize terminal` message rather than corrupting layout.
- **Bubbletea startup error**: `p.Run()` returns an `error`. Always check it.
  On error, log to stderr at `debug` level and re-run the command in plain mode
  rather than surfacing a raw bubbletea stack trace to the user.
- **Panic recovery**: wrap `p.Run()` in a deferred recover. On panic, restore the
  terminal (`tea.ClearScreen`, `fmt.Print(cursor.Show)`) before re-panicking or
  returning a clean error — a crashed TUI must never leave the terminal in raw mode.
- **Interrupt / SIGTERM**: bubbletea handles `ctrl+c` internally via `tea.Quit`.
  Rudder must not install its own signal handler that races with bubbletea's.

```go
// cmd/rudder/use.go
func runUsePicker(app *App) error {
    if !term.IsTerminal(int(os.Stdout.Fd())) || app.NoTUI {
        return runUsePlain(app)
    }
    m := ui.NewPickerModel(app.Config.Environments, app.Theme)
    p := tea.NewProgram(m, tea.WithAltScreen())
    defer func() {
        if r := recover(); r != nil {
            p.ReleaseTerminal()
            panic(r) // re-panic with terminal restored
        }
    }()
    result, err := p.Run()
    if err != nil {
        app.Log.Debug("bubbletea error, falling back to plain", "err", err)
        return runUsePlain(app)
    }
    // extract selected env from result model...
}
```

### Status Indicators

```
● dev        kind-dev          [local, kind]        ✓ reachable
● staging    aks-staging-admin [azure, aks]         ✓ reachable
● prod       aks-prod-admin    [azure, aks, prod]   ✗ unreachable
◌ dr         aks-dr-admin      [azure, aks]           probing...
```

---

## Performance Architecture

### Startup Performance
- Config is loaded lazily in `PersistentPreRun` only for commands that need it
- `rudder use <env>` (direct, no picker) writes `state.yaml` and exits in <10ms
- No kubeconfig parsing on simple passthrough commands — only path/context
  are read from Rudder config; `client-go` merge is never invoked for passthrough

### Concurrent Status Probing
Cluster reachability checks in the fuzzy picker run concurrently via `errgroup`
with a per-probe timeout (default 2s). Results stream back into the bubbletea
model via `tea.Cmd` messages — the picker is never blocked waiting for slow clusters.

```go
// internal/cluster/ping.go
// Lightweight reachability: hits /readyz on the API server via the kubeconfig's
// server URL. Does NOT use client-go to avoid the overhead of full client init.
func Ping(ctx context.Context, kubeconfigPath, contextName string) Status
```

### kubectl Passthrough Performance
- `kubectl` is resolved once and cached on the `App` struct
- `exec.Cmd` is constructed with minimal overhead — no shell, direct exec
- `KUBECONFIG` and `--context` are the only injections; no kubeconfig rewriting

### Fan-out (`--all`)
- Environments execute **sequentially** by design — predictable, readable output
- Each execution streams directly to stdout under a styled header
- A per-cluster timeout can be set via `--timeout` flag (default: none)

---

## Key Dependencies

| Package                              | Purpose                                      |
|--------------------------------------|----------------------------------------------|
| `github.com/spf13/cobra`             | CLI command tree                             |
| `github.com/spf13/viper`             | Config file loading / env vars               |
| `k8s.io/client-go`                   | Kubeconfig loading & merging                 |
| `github.com/charmbracelet/bubbletea` | Interactive TUI                              |
| `github.com/charmbracelet/bubbles`   | Spinner, table, list, text input components  |
| `github.com/charmbracelet/lipgloss`  | Adaptive terminal styling                    |
| `github.com/charmbracelet/glamour`   | Markdown rendering                           |
| `github.com/muesli/termenv`          | Terminal background/capability detection     |
| `golang.org/x/sync/errgroup`         | Concurrent status probing                    |
| `gopkg.in/yaml.v3`                   | YAML config marshaling                       |

---

## Environment Variables

| Variable           | Description                                              |
|--------------------|----------------------------------------------------------|
| `RUDDER_CONFIG`    | Override path to `config.yaml` (default: `~/.rudder/`)  |
| `RUDDER_ENV`       | Override active environment for a single invocation     |
| `RUDDER_KUBECTL`   | Path to kubectl binary                                   |
| `RUDDER_NO_COLOR`  | Disable color output                                     |
| `RUDDER_NO_TUI`    | Disable interactive TUI (force plain output)             |
| `RUDDER_LOG_LEVEL` | Log verbosity: `debug`, `info`, `warn`, `error`          |
| `RUDDER_CI`        | Set to `true` in CI — implies NO_TUI and NO_COLOR        |

`RUDDER_ENV` takes precedence over `state.yaml` for one-shot overrides:
```sh
RUDDER_ENV=prod rudder exec -- get pods -n platform
```

---

## Development

### Prerequisites
- Go 1.22+
- `kubectl` available on PATH (or set `RUDDER_KUBECTL`)
- `golangci-lint` for linting
- `goreleaser` for release builds
- `just` for task running

### Justfile

```just
# Default recipe — list all available recipes
default:
    @just --list

# Build the rudder binary
build:
    go build -ldflags="-X github.com/yourusername/rudder/pkg/rudder.Version={{version}}" \
        -o bin/rudder ./cmd/rudder

# Install to $GOPATH/bin
install:
    go install ./cmd/rudder

# Run all tests
test:
    go test ./...

# Run tests with coverage report
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Lint with golangci-lint
lint:
    golangci-lint run ./...

# Format code (gofmt + goimports)
fmt:
    gofmt -w .
    goimports -w .

# GoReleaser snapshot build (no publish)
release-dry:
    goreleaser release --snapshot --clean

# Remove build artifacts
clean:
    rm -rf bin/ dist/ coverage.out completions/

# Generate shell completions
completions:
    mkdir -p completions
    go run ./cmd/rudder completion bash > completions/rudder.bash
    go run ./cmd/rudder completion zsh > completions/rudder.zsh
    go run ./cmd/rudder completion fish > completions/rudder.fish

# Run rudder locally (pass args: `just run use`, `just run envs`)
run *args:
    go run ./cmd/rudder {{args}}

# Tidy go modules
tidy:
    go mod tidy

# Run fmt, lint, and test (pre-commit / CI equivalent)
check: fmt lint test
```

### Running Locally

```sh
just run envs
just run use dev
just run exec -- get nodes
```

---

## Code Conventions

- **Package names**: lowercase, single word, no underscores.
- **Error handling**: always wrap with `fmt.Errorf("context: %w", err)`. Never swallow errors.
- **Config loading**: done once in `PersistentPreRun` on the root command; stored on a shared
  `App` struct passed into `cobra.Command.RunE` via closure.
- **No globals**: avoid package-level mutable state. Pass `App` and `Theme` explicitly.
- **Kubeconfig paths**: always expand `~` and env vars before any filesystem operations.
  Expansion lives in `internal/config/loader.go`.
- **Logging**: use `log/slog` (stdlib). No `fmt.Println` outside of `internal/ui`.
- **Output**: all user-facing output goes through `internal/ui/output.go`. Raw `fmt.Fprintf`
  to stdout/stderr only in `output.go`.
- **kubectl exec**: always use `exec.Cmd.Env` for `KUBECONFIG` injection — never `os.Setenv`.
  The env var must be scoped to the subprocess only.
- **Tests**: table-driven tests. Use `testdata/` directories for fixture kubeconfigs.
- **TUI graceful degradation**: all bubbletea models must check `RUDDER_NO_TUI` / `RUDDER_CI`
  and fall back to plain output. No interactive prompt should ever block a CI pipeline.

---

## Testing Strategy

| Layer              | Approach                                                        |
|--------------------|-----------------------------------------------------------------|
| Config loading     | Unit tests with temp dirs and fixture YAML files                |
| Kubeconfig merge   | Unit tests using `client-go` fake kubeconfig structs            |
| Cluster ping       | Unit tests with mock HTTP server returning `/readyz` responses  |
| kubectl exec       | Integration tests with a mock `kubectl` script on PATH          |
| CLI commands       | `cobra` command tests using `Execute()` with captured stdout    |
| UI/prompts         | Skipped in CI (`RUDDER_CI=true`); bubbletea models unit tested  |

---

## Shell Completions

Generate completions via `just completions`. The output lands in `completions/`
(gitignored). Cobra's built-in `completion` command also works directly:
`rudder completion bash|zsh|fish|powershell`.

---

## Roadmap

Rudder's roadmap is tracked as GitHub Issues with the following labels:

| Label            | Meaning                                      |
|------------------|----------------------------------------------|
| `enhancement`    | New feature or improvement                   |
| `ux`             | Visual/interaction improvements              |
| `performance`    | Startup time, exec latency, resource usage   |
| `breaking`       | Requires config/API migration                |

See [Work Items](https://gitlab.com/dcresp0/rudder/-/work_items)
for the current backlog. Do not maintain a duplicate checklist here — it will go stale.

---

## Notes for Claude

- `~/.rudder/state.yaml` must be written **atomically** using a rename-on-write
  pattern to be safe under concurrent `rudder use` invocations (e.g. two terminal
  tabs switching envs simultaneously). Write to a temp file in the same directory
  (`~/.rudder/state.yaml.tmp`), then `os.Rename()` — which is atomic on POSIX
  filesystems. Never write directly to `state.yaml` with `os.WriteFile`.

```go
// internal/config/loader.go
func SaveState(dir string, state *State) error {
    data, err := yaml.Marshal(state)
    if err != nil {
        return fmt.Errorf("marshal state: %w", err)
    }
    tmp := filepath.Join(dir, "state.yaml.tmp")
    if err := os.WriteFile(tmp, data, 0600); err != nil {
        return fmt.Errorf("write temp state: %w", err)
    }
    if err := os.Rename(tmp, filepath.Join(dir, "state.yaml")); err != nil {
        return fmt.Errorf("rename state: %w", err)
    }
    return nil
}
```

- The core passthrough pattern lives in `internal/kubectl/exec.go`. When exec-ing
  kubectl, always inject both `KUBECONFIG` env var AND `--context` flag — never rely
  on kubeconfig `current-context` mutation, which is a side effect that leaks state.
- The Rudder config (`~/.rudder/config.yaml`) is **never** mutated by `rudder use` —
  only `~/.rudder/state.yaml` is written on context switch. The environment registry
  stays stable and git-trackable.
- Kubeconfig paths support `~` expansion and env vars (e.g. `$KUBECONFIG_DIR/dev.yaml`).
  Always expand in `internal/config/loader.go` before any filesystem operations.
- The `Theme` struct must be constructed once and passed down — do not call `NewTheme()`
  inside bubbletea `Update` or `View` functions as terminal detection has overhead.
- Cluster ping in `internal/cluster/ping.go` must use a short-lived `http.Client` with
  explicit timeout — never the default client. Reuse the kubeconfig's TLS config but
  set `Timeout: 2 * time.Second` on the client.
- For GCC High / Azure Government contexts, kubeconfig server URLs point to `*.azmk8s.us`
  — no special handling needed in Rudder, but worth noting in environment descriptions.
- `RUDDER_CI=true` must imply both `RUDDER_NO_COLOR` and `RUDDER_NO_TUI`. Check this
  once in `PersistentPreRun` and set both flags on the `App` struct.
- All bubbletea models live in `internal/ui/` and must implement a `PlainView() string`
  method used as fallback when TUI is disabled.