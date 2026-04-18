# Environments & Kubeconfig Management

## What is an environment?

An environment is a named alias for a Kubernetes cluster context. It maps to:

- A path to a kubeconfig file
- An optional context name within that kubeconfig
- Optional metadata: description, tags, namespace override

Environments are stored in `~/.rudder/config.yaml`. This file is stable and
git-trackable — it is never mutated by `rudder use` or `rudder exec`. Only
`~/.rudder/state.yaml` changes when you switch environments.

## Kubeconfig paths

Kubeconfig paths in `config.yaml` support:

- **Tilde expansion**: `~/` is expanded to your home directory
- **Environment variables**: `$KUBECONFIG_DIR/dev.yaml` is fully expanded

Expansion happens in `internal/config/loader.go` before any filesystem
operations. The raw (unexpanded) path is stored in `config.yaml`, so the
config remains portable across machines with different home directories.

```yaml
environments:
  - name: dev
    kubeconfig: ~/.kube/kind-dev.yaml      # ~ expanded at runtime
  - name: staging
    kubeconfig: $KUBECONFIG_DIR/staging.yaml  # env var expanded at runtime
```

## Context selection

If `context` is omitted in the environment definition, Rudder uses the
kubeconfig's `current-context` field at execution time. **It does not mutate
`current-context`** — it always passes `--context=<name>` explicitly to kubectl.

```yaml
environments:
  - name: dev
    kubeconfig: ~/.kube/dev.yaml
    # context omitted — uses current-context from the kubeconfig file

  - name: staging
    kubeconfig: ~/.kube/staging.yaml
    context: aks-staging-admin  # explicit — ignores current-context
```

## The active environment

The active environment is stored in `~/.rudder/state.yaml`:

```yaml
active_environment: staging
```

This file is written atomically (rename-on-write) to be safe under concurrent
`rudder use` invocations (e.g. two terminal tabs switching environments at once).

**No ambient kubeconfig fallback.** If `state.yaml` is absent or empty and
`RUDDER_ENV` is not set, `rudder exec` exits with an error rather than silently
falling back to `~/.kube/config`. This prevents accidental operations against
an unintended cluster.

## One-shot environment override

`RUDDER_ENV` overrides the persisted active environment for a single invocation:

```sh
RUDDER_ENV=prod rudder exec -- get nodes
RUDDER_ENV=prod rudder envs   # shows prod as active
```

`RUDDER_ENV` takes precedence over `state.yaml`. It does not modify `state.yaml`.

## Tags

Tags are free-form labels for grouping and filtering environments:

```yaml
environments:
  - name: staging
    tags: [azure, aks, govcloud]
  - name: prod
    tags: [azure, aks, govcloud, prod]
```

Filter with `--tags` on `rudder envs` and `rudder exec --all`:

```sh
rudder envs --tags govcloud
rudder exec --all --tags prod -- get nodes
```

Multiple tags are ANDed — an environment must have **all** specified tags to match.

## Namespace overrides

A `namespace` field on an environment prepends `--namespace=<ns>` to all kubectl
invocations for that environment:

```yaml
environments:
  - name: staging
    kubeconfig: ~/.kube/staging.yaml
    namespace: platform   # --namespace=platform prepended to all exec calls
```

The global `default_namespace` in `config.yaml` applies to environments that
have no per-environment `namespace` set (not yet implemented — tracked as an enhancement).

## Multiple kubeconfigs

Rudder does not merge kubeconfigs by default. Each environment points to exactly
one kubeconfig file. If you have contexts spread across multiple kubeconfigs,
register each as a separate environment:

```yaml
environments:
  - name: dev
    kubeconfig: ~/.kube/kind-dev.yaml
    context: kind-dev
  - name: staging
    kubeconfig: ~/.kube/aks-staging.yaml
    context: aks-staging-admin
```

`rudder kubeconfig export` (planned) will merge and print a unified kubeconfig.

## Azure Government (GCC High)

Kubeconfig server URLs for Azure Government clusters point to `*.azmk8s.us`
instead of `*.azmk8s.io`. No special Rudder configuration is needed — the URL
comes from the kubeconfig itself. Annotating the environment with a tag helps
with filtering:

```yaml
environments:
  - name: prod-gcch
    kubeconfig: ~/.kube/aks-prod-gcch.yaml
    context: aks-prod-gcch-admin
    tags: [azure, aks, govcloud, gcc-high, prod]
```
