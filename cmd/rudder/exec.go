package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dcrespo1/rudder/internal/config"
	"github.com/dcrespo1/rudder/internal/kubectl"
	"github.com/dcrespo1/rudder/internal/ui"
)

func NewExecCmd(app *App) *cobra.Command {
	var all bool
	var tags []string
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "exec -- [kubectl args]",
		Short: "Run kubectl against the active environment",
		Long: `Run kubectl against the active environment.

The active environment's kubeconfig is injected as KUBECONFIG=<path> and
--context=<context> is prepended automatically. KUBECONFIG is never leaked
to the parent process — it is scoped to the kubectl subprocess only.

If no active environment is set and RUDDER_ENV is not provided, rudder exits
with an error rather than silently falling back to the ambient kubeconfig.

Use -- to separate rudder flags from kubectl arguments:
  rudder exec -- get pods -n kube-system

With --all, the command runs sequentially across all (or filtered) environments:
  rudder exec --all -- get nodes
  rudder exec --all --tags govcloud -- get pods -n platform`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Strip leading "--" separator if present
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}

			ctx := context.Background()

			if all {
				return runExecAll(ctx, app, tags, timeout, args)
			}
			return runExecSingle(ctx, app, args)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "run across all registered environments sequentially")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "filter environments by tag (only with --all)")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "per-cluster timeout (e.g. 30s); 0 means no limit (only with --all)")

	return cmd
}

// runExecSingle runs kubectl against the single active environment.
func runExecSingle(ctx context.Context, app *App, args []string) error {
	env, err := resolveActiveEnv(app)
	if err != nil {
		return err
	}

	kubeconfigPath := config.ExpandPath(env.Kubeconfig)
	contextName := env.Context

	if err := kubectl.Run(ctx, app.KubectlPath, kubeconfigPath, contextName, env.Namespace, args, app.Stdout, app.Stderr); err != nil {
		// Propagate kubectl's exit code without wrapping — don't add noise
		// on top of kubectl's own error output.
		if isExitError(err) {
			return &silentError{err}
		}
		return err
	}
	return nil
}

// runExecAll fans out kubectl across multiple environments sequentially.
func runExecAll(ctx context.Context, app *App, tags []string, timeout time.Duration, args []string) error {
	envs := app.Config.FilterByTags(tags)
	if len(envs) == 0 {
		ui.PrintWarning(app.Stdout, app.Theme, "no environments match the specified tags")
		return nil
	}
	return kubectl.RunFanOut(ctx, envs, args, app.KubectlPath, timeout, app.Stdout, app.Theme)
}

// resolveActiveEnv determines the active environment.
// Resolution order: RUDDER_ENV env var → state.yaml → error.
func resolveActiveEnv(app *App) (*config.Environment, error) {
	name := os.Getenv("RUDDER_ENV")
	if name == "" && app.State != nil {
		name = app.State.ActiveEnvironment
	}
	if name == "" {
		return nil, fmt.Errorf("no active environment set — run `rudder use` to select one")
	}

	env, ok := app.Config.FindEnv(name)
	if !ok {
		return nil, fmt.Errorf("active environment %q not found in config — run `rudder envs` to list available environments", name)
	}
	return env, nil
}

func isExitError(err error) bool {
	type exitCoder interface{ ExitCode() int }
	if _, ok := err.(exitCoder); ok {
		return true
	}
	// *exec.ExitError wraps via %w; check the unwrapped chain
	unwrapped := err
	for unwrapped != nil {
		if _, ok := unwrapped.(exitCoder); ok {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := unwrapped.(unwrapper)
		if !ok {
			break
		}
		unwrapped = u.Unwrap()
	}
	return false
}

// silentError is an error that produces no additional output — the subprocess
// already printed its own error message. The non-zero exit code is propagated
// by main.go via os.Exit.
type silentError struct{ cause error }

func (e *silentError) Error() string { return "" }
func (e *silentError) Unwrap() error { return e.cause }
