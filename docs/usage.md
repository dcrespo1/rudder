# Command Reference

## Global flags

These flags are available on every command:

| Flag              | Description                                    |
|-------------------|------------------------------------------------|
| `--config-dir`    | Override config directory (default: `~/.rudder`) |
| `--no-color`      | Disable color output                           |
| `--no-tui`        | Disable interactive TUI, force plain output    |
| `--log-level`     | Log verbosity: `debug`, `info`, `warn`, `error`|
| `-h`, `--help`    | Help for any command                           |

---

## `rudder init`

Interactive first-time setup wizard. Scans for existing kubeconfigs, prompts you
to register environments, and writes `~/.rudder/config.yaml`.

```sh
rudder init
```

If a config already exists, you will be prompted before overwriting it.

---

## `rudder envs`

List all registered environments with status indicators.

```sh
rudder envs
rudder envs --tags govcloud           # filter by tag
rudder envs --output json             # JSON output
```

**Flags:**

| Flag       | Description                                       |
|------------|---------------------------------------------------|
| `--tags`   | Comma-separated tags to filter by                 |
| `-o`       | Output format: `table` (default) or `json`        |

**Output columns:** active marker, name, context, namespace, tags

The `●` marker in the first column indicates the currently active environment.

---

## `rudder use [env]`

Switch the active environment.

```sh
rudder use              # opens fuzzy picker with live cluster status
rudder use staging      # direct switch, no picker
```

**Picker behavior:**
- Environments load instantly; reachability status (`●`/`●`/`◌`) populates async
- Fuzzy search filters by name, description, and tags
- `↑`/`↓` or `j`/`k` to navigate, `Enter` to select, `Esc`/`q` to cancel
- Disabled automatically when stdout is not a TTY or `RUDDER_NO_TUI=true`

The switch writes `~/.rudder/state.yaml` atomically. The environment registry
(`config.yaml`) is never mutated by `rudder use`.

---

## `rudder exec`

Run `kubectl` against the active environment.

```sh
# Explicit exec subcommand — use -- to separate rudder flags from kubectl args
rudder exec -- get pods -n kube-system
rudder exec -- apply -f deployment.yaml

# One-shot environment override
RUDDER_ENV=prod rudder exec -- get nodes

# Fan-out across all environments
rudder exec --all -- get nodes
rudder exec --all --tags govcloud -- get pods -n platform
rudder exec --all --timeout 30s -- get nodes
```

**Flags:**

| Flag        | Description                                                  |
|-------------|--------------------------------------------------------------|
| `--all`     | Run sequentially across all (or filtered) environments       |
| `--tags`    | Filter environments by tag (only with `--all`)               |
| `--timeout` | Per-cluster timeout e.g. `30s` (only with `--all`; default: none) |

**Injection:** Rudder injects `KUBECONFIG=<path>` into the subprocess environment
and prepends `--context=<ctx>` to the kubectl arguments. The `KUBECONFIG` variable
is **never** set on the parent process.

**No active environment:** If no environment is set and `RUDDER_ENV` is not
provided, `rudder exec` exits with:
```
error: no active environment set — run `rudder use` to select one
```

**Fan-out output format:**
```
─── staging ───────────────────────────────────────────
NAME             STATUS   ROLES   AGE
aks-node-001     Ready    agent   12d

─── prod ──────────────────────────────────────────────
NAME             STATUS   ROLES   AGE
aks-node-001     Ready    agent   45d
```

---

## `rudder config`

Manage the environment registry.

### `rudder config add`

```sh
# Interactive wizard
rudder config add

# Non-interactive with flags
rudder config add \
  --name staging \
  --kubeconfig ~/.kube/aks-staging.yaml \
  --context aks-staging-admin \
  --namespace platform \
  --tags azure,aks,govcloud \
  --description "Azure GovCloud staging AKS"
```

**Flags:**

| Flag            | Description                                       |
|-----------------|---------------------------------------------------|
| `--name`        | Environment name (required for non-interactive)   |
| `--kubeconfig`  | Path to kubeconfig file                           |
| `--context`     | Context name (default: kubeconfig current-context)|
| `--namespace`   | Default namespace override                        |
| `--tags`        | Comma-separated tags                              |
| `--description` | Human-readable description                        |

### `rudder config remove <name>`

```sh
rudder config remove staging
rudder config remove staging --yes   # skip confirmation
```

If the removed environment was active, the active environment is cleared.

### `rudder config rename <old> <new>`

```sh
rudder config rename staging aks-staging
```

If the renamed environment was active, `state.yaml` is updated automatically.

### `rudder config list`

```sh
rudder config list
```

Same output as `rudder envs` but always shows all environments without status probing.

### `rudder config edit`

```sh
rudder config edit
```

Opens `config.yaml` in `$EDITOR` (falls back to `$VISUAL`, then `vi`).

### `rudder config validate`

```sh
rudder config validate
```

Checks all registered environments:
- Kubeconfig file exists and is a regular file
- Environment name contains only `[a-z0-9-_]`

Exits non-zero if any environment fails validation.

---

## `rudder version`

```sh
rudder version          # full build info
rudder version --short  # version string only
```
