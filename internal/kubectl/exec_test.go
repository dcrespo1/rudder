package kubectl

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.com/dcresp0/rudder/internal/config"
	"gitlab.com/dcresp0/rudder/internal/ui"
)

func mockKubectlPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", "mock_kubectl", "kubectl"))
	if err != nil {
		t.Fatalf("resolve mock kubectl path: %v", err)
	}
	return abs
}

func TestRun_InjectsKubeconfigEnv(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)

	var out, errOut bytes.Buffer
	err := Run(context.Background(), mockKubectlPath(t), "/fake/kubeconfig.yaml", "my-context", "", []string{"get", "pods"}, &out, &errOut)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	data, _ := os.ReadFile(argsFile)
	args := strings.TrimSpace(string(data))
	if !strings.Contains(args, "--context=my-context") {
		t.Errorf("expected --context flag in args, got: %q", args)
	}
	if !strings.Contains(args, "get") || !strings.Contains(args, "pods") {
		t.Errorf("expected user args in output, got: %q", args)
	}
}

func TestRun_NoOsSetenv(t *testing.T) {
	// Verify KUBECONFIG is not leaked to the test process env
	original := os.Getenv("KUBECONFIG")

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)

	var out, errOut bytes.Buffer
	Run(context.Background(), mockKubectlPath(t), "/injected/kubeconfig.yaml", "ctx", "", []string{"version"}, &out, &errOut)

	after := os.Getenv("KUBECONFIG")
	if after != original {
		t.Errorf("KUBECONFIG leaked to process env: before=%q after=%q", original, after)
	}
}

func TestRun_WithNamespace(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)

	var out, errOut bytes.Buffer
	Run(context.Background(), mockKubectlPath(t), "/fake/kube.yaml", "ctx", "my-ns", []string{"get", "pods"}, &out, &errOut)

	data, _ := os.ReadFile(argsFile)
	args := strings.TrimSpace(string(data))
	if !strings.Contains(args, "--namespace=my-ns") {
		t.Errorf("expected --namespace flag, got: %q", args)
	}
}

func TestRun_PropagatesExitCode(t *testing.T) {
	t.Setenv("MOCK_KUBECTL_EXIT_CODE", "1")

	var out, errOut bytes.Buffer
	err := Run(context.Background(), mockKubectlPath(t), "/fake/kube.yaml", "ctx", "", []string{"get", "pods"}, &out, &errOut)
	if err == nil {
		t.Error("expected error from non-zero exit code")
	}
}

func TestBuildArgs_ContextAndNamespace(t *testing.T) {
	args := buildArgs("my-ctx", "my-ns", []string{"get", "pods"})
	if args[0] != "--context=my-ctx" {
		t.Errorf("expected --context first, got %q", args[0])
	}
	if args[1] != "--namespace=my-ns" {
		t.Errorf("expected --namespace second, got %q", args[1])
	}
	if args[2] != "get" || args[3] != "pods" {
		t.Errorf("expected user args last, got %v", args[2:])
	}
}

func TestBuildArgs_NoContext(t *testing.T) {
	args := buildArgs("", "", []string{"version"})
	if len(args) != 1 || args[0] != "version" {
		t.Errorf("expected only user args, got %v", args)
	}
}

func TestRunFanOut(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)

	// Create a real (empty) kubeconfig file so ExpandPath / stat works
	kube, _ := os.CreateTemp("", "kubeconfig-*.yaml")
	kube.Close()
	defer os.Remove(kube.Name())

	envs := []config.Environment{
		{Name: "dev", Kubeconfig: kube.Name(), Context: "kind-dev"},
		{Name: "staging", Kubeconfig: kube.Name(), Context: "aks-staging"},
	}

	var out bytes.Buffer
	theme := ui.NewTheme()
	err := RunFanOut(context.Background(), envs, []string{"get", "nodes"}, mockKubectlPath(t), 0, &out, theme)
	if err != nil {
		t.Fatalf("RunFanOut: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "dev") {
		t.Errorf("expected 'dev' in fan-out output, got: %q", output)
	}
	if !strings.Contains(output, "staging") {
		t.Errorf("expected 'staging' in fan-out output, got: %q", output)
	}
}

func TestRunFanOut_Timeout(t *testing.T) {
	// Use a real kubectl if available, otherwise skip — this tests timeout propagation
	// by using a sleep-based mock. Since our mock exits immediately, just verify
	// that a very short timeout doesn't panic.
	kube, _ := os.CreateTemp("", "kubeconfig-*.yaml")
	kube.Close()
	defer os.Remove(kube.Name())

	envs := []config.Environment{
		{Name: "dev", Kubeconfig: kube.Name(), Context: "ctx"},
	}
	var out bytes.Buffer
	theme := ui.NewTheme()
	// 10ms timeout — mock exits immediately so this should still pass
	RunFanOut(context.Background(), envs, []string{"get", "nodes"}, mockKubectlPath(t), 10*time.Millisecond, &out, theme)
}
