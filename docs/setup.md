# Setup & Installation

## Prerequisites

- Go 1.22 or later (for `go install`)
- `kubectl` on your PATH (or set `RUDDER_KUBECTL`)

## Installation

### via `go install`

```sh
go install gitlab.com/dcresp0/rudder/cmd/rudder@latest
```

### via the install script

```sh
curl -fsSL https://gitlab.com/dcresp0/rudder/-/raw/main/scripts/install.sh | sh
```

The script detects your OS and architecture, downloads the appropriate release
archive from GitLab Releases, and installs the binary to `/usr/local/bin`
(or `$HOME/.local/bin` if `/usr/local/bin` is not writable).

### via release archive

Download the appropriate archive from the
[Releases page](https://gitlab.com/dcresp0/rudder/-/releases), extract it,
and place the `rudder` binary on your PATH.

## First-time setup

After installation, run the setup wizard:

```sh
rudder init
```

The wizard scans for existing kubeconfigs (`~/.kube/config`, `$KUBECONFIG`),
lets you register environments interactively, and writes `~/.rudder/config.yaml`.

If you prefer to set up manually, copy the example config:

```sh
mkdir -p ~/.rudder
cp "$(go env GOPATH)/pkg/mod/gitlab.com/dcresp0/rudder@latest/configs/config.example.yaml" \
   ~/.rudder/config.yaml
# Edit ~/.rudder/config.yaml to reflect your clusters
```

## Select your active environment

```sh
rudder use          # opens the fuzzy picker
rudder use staging  # direct switch
```

## Shell completions

```sh
# Bash
rudder completion bash > /etc/bash_completion.d/rudder

# Zsh
rudder completion zsh > "${fpath[1]}/_rudder"

# Fish
rudder completion fish > ~/.config/fish/completions/rudder.fish
```

## Verifying the installation

```sh
rudder version
rudder envs
```

## Environment variables

| Variable           | Description                                              |
|--------------------|----------------------------------------------------------|
| `RUDDER_CONFIG`    | Override path to config directory (default: `~/.rudder`)|
| `RUDDER_ENV`       | Override active environment for a single invocation     |
| `RUDDER_KUBECTL`   | Path to kubectl binary                                   |
| `RUDDER_NO_COLOR`  | Disable color output                                     |
| `RUDDER_NO_TUI`    | Disable interactive TUI (force plain output)             |
| `RUDDER_LOG_LEVEL` | Log verbosity: `debug`, `info`, `warn`, `error`          |
| `RUDDER_CI`        | Set to `true` in CI — implies `NO_TUI` and `NO_COLOR`   |

## Upgrading

```sh
go install gitlab.com/dcresp0/rudder/cmd/rudder@latest
```

Or re-run the install script. Your `~/.rudder/config.yaml` is never modified
by upgrades.
