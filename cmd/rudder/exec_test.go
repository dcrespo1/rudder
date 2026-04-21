package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dcrespo1/rudder/internal/config"
)

// setupExecTest creates a config dir, kubeconfig, registered env, and active state.
// Returns the config dir path.
func setupExecTest(t *testing.T, envName, contextName string) string {
	t.Helper()
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, envName, "https://127.0.0.1:6443", contextName)
	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: envName, Kubeconfig: kube, Context: contextName},
		},
	})
	writeState(t, dir, envName)
	return dir
}

func TestExec_InjectsContext(t *testing.T) {
	dir := setupExecTest(t, "dev", "kind-dev")

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))

	mustRunCmd(t, dir, "exec", "--", "get", "pods")

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	args := strings.TrimSpace(string(data))
	if !strings.Contains(args, "--context=kind-dev") {
		t.Errorf("expected --context=kind-dev in args, got: %q", args)
	}
	if !strings.Contains(args, "get") || !strings.Contains(args, "pods") {
		t.Errorf("expected user args in output, got: %q", args)
	}
}

func TestExec_KubeconfigNotLeakedToProcess(t *testing.T) {
	dir := setupExecTest(t, "dev", "kind-dev")

	original := os.Getenv("KUBECONFIG")
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))

	runCmd(t, dir, "exec", "--", "version")

	// KUBECONFIG must not have changed in the test process
	if after := os.Getenv("KUBECONFIG"); after != original {
		t.Errorf("KUBECONFIG leaked to process env: was %q, now %q", original, after)
	}
}

func TestExec_NoActiveEnv_Error(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "dev", "https://127.0.0.1:6443", "ctx")
	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})
	// No state.yaml — no active environment

	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))
	t.Setenv("RUDDER_ENV", "") // ensure override is not set

	r := runCmd(t, dir, "exec", "--", "get", "pods")
	if r.Err == nil {
		t.Error("expected error when no active environment is set")
	}
	if !strings.Contains(r.Err.Error(), "no active environment") {
		t.Errorf("expected 'no active environment' error, got: %v", r.Err)
	}
}

func TestExec_RudderEnvOverride(t *testing.T) {
	dir := t.TempDir()
	kubeA := writeKubeconfig(t, dir, "dev", "https://127.0.0.1:6443", "kind-dev")
	kubeB := writeKubeconfig(t, dir, "prod", "https://127.0.0.1:6443", "aks-prod")
	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kubeA, Context: "kind-dev"},
			{Name: "prod", Kubeconfig: kubeB, Context: "aks-prod"},
		},
	})
	writeState(t, dir, "dev") // persisted active = dev
	t.Setenv("RUDDER_ENV", "prod")
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))

	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("MOCK_KUBECTL_ARGS_FILE", argsFile)

	mustRunCmd(t, dir, "exec", "--", "get", "nodes")

	data, _ := os.ReadFile(argsFile)
	args := strings.TrimSpace(string(data))
	if !strings.Contains(args, "--context=aks-prod") {
		t.Errorf("expected prod context via RUDDER_ENV override, got: %q", args)
	}
}

func TestExec_ExitCodePropagated(t *testing.T) {
	dir := setupExecTest(t, "dev", "ctx")
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))
	t.Setenv("MOCK_KUBECTL_EXIT_CODE", "2")

	r := runCmd(t, dir, "exec", "--", "get", "pods")
	if r.Err == nil {
		t.Error("expected error when kubectl exits non-zero")
	}
}

func TestExec_All(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")
	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube, Context: "ctx"},
			{Name: "staging", Kubeconfig: kube, Context: "ctx"},
		},
	})
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))

	r := mustRunCmd(t, dir, "exec", "--all", "--", "get", "nodes")

	if !strings.Contains(r.Stdout, "dev") {
		t.Errorf("expected 'dev' header in fan-out output, got: %q", r.Stdout)
	}
	if !strings.Contains(r.Stdout, "staging") {
		t.Errorf("expected 'staging' header in fan-out output, got: %q", r.Stdout)
	}
}

func TestExec_All_TagFilter(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")
	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube, Tags: []string{"local"}},
			{Name: "prod", Kubeconfig: kube, Tags: []string{"azure", "prod"}},
		},
	})
	t.Setenv("RUDDER_KUBECTL", mockKubectlPath(t))

	r := mustRunCmd(t, dir, "exec", "--all", "--tags", "azure", "--", "get", "nodes")

	if strings.Contains(r.Stdout, "dev") {
		t.Error("'dev' should be filtered out by --tags azure")
	}
	if !strings.Contains(r.Stdout, "prod") {
		t.Error("expected 'prod' in filtered fan-out output")
	}
}
