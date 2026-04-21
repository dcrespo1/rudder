package kubectl

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"gitlab.com/dcresp0/rudder/internal/config"
	"gitlab.com/dcresp0/rudder/internal/ui"
)

// Run executes kubectl with the given kubeconfig and context injected.
// KUBECONFIG is scoped to the subprocess via cmd.Env — never os.Setenv.
// --context is prepended to args so it always takes effect.
func Run(ctx context.Context, kubectlPath, kubeconfigPath, contextName, namespace string, args []string, stdout, stderr io.Writer) error {
	fullArgs := buildArgs(contextName, namespace, args)

	cmd := exec.CommandContext(ctx, kubectlPath, fullArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl: %w", err)
	}
	return nil
}

// RunFanOut executes kubectl sequentially across multiple environments.
// Output for each env is preceded by a styled cluster header.
// If timeout > 0, each individual execution is bounded by that timeout.
func RunFanOut(ctx context.Context, envs []config.Environment, args []string, kubectlPath string, timeout time.Duration, out io.Writer, theme ui.Theme) error {
	for _, env := range envs {
		ui.PrintClusterHeader(out, theme, env.Name)

		execCtx := ctx
		var cancel context.CancelFunc
		if timeout > 0 {
			execCtx, cancel = context.WithTimeout(ctx, timeout)
		}

		kubeconfigPath := config.ExpandPath(env.Kubeconfig)
		contextName := env.Context
		if contextName == "" {
			contextName = ""
		}

		err := Run(execCtx, kubectlPath, kubeconfigPath, contextName, env.Namespace, args, out, os.Stderr)
		if cancel != nil {
			cancel()
		}
		if err != nil {
			fmt.Fprintf(out, "error: %v\n\n", err)
			continue
		}
		fmt.Fprintln(out)
	}
	return nil
}

// buildArgs prepends --context and optionally --namespace to the user-supplied args.
func buildArgs(contextName, namespace string, args []string) []string {
	var prefix []string
	if contextName != "" {
		prefix = append(prefix, "--context="+contextName)
	}
	if namespace != "" {
		prefix = append(prefix, "--namespace="+namespace)
	}
	return append(prefix, args...)
}
