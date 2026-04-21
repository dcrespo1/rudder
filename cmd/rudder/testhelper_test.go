package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dcrespo1/rudder/internal/config"
)

// cmdResult holds the output of a test command execution.
type cmdResult struct {
	Stdout string
	Stderr string
	Err    error
}

// runCmd builds a fresh command tree and executes args with configDir injected
// as --config-dir and --no-tui/--no-color flags prepended.
//
// pflag.StringVar/BoolVar reset the bound variable to the flag's default value
// when the flag set is registered, so we can't pre-set App fields before calling
// newCommandTree. Instead, we pass configDir as a flag argument so cobra applies
// it during normal flag parsing.
func runCmd(t *testing.T, configDir string, args ...string) cmdResult {
	t.Helper()

	app := NewApp()
	root := newCommandTree(app)

	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)

	// Prepend config dir and CI-friendly flags; user args follow
	fullArgs := append([]string{"--config-dir", configDir, "--no-tui", "--no-color"}, args...)
	root.SetArgs(fullArgs)

	err := root.Execute()
	return cmdResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}

// mustRunCmd calls runCmd and fails the test if it returns an error.
func mustRunCmd(t *testing.T, configDir string, args ...string) cmdResult {
	t.Helper()
	r := runCmd(t, configDir, args...)
	if r.Err != nil {
		t.Fatalf("command %v failed: %v\nstdout: %s\nstderr: %s", args, r.Err, r.Stdout, r.Stderr)
	}
	return r
}

// writeKubeconfig writes a minimal kubeconfig to dir/name.yaml and returns the path.
func writeKubeconfig(t *testing.T, dir, name, serverURL, contextName string) string {
	t.Helper()
	content := "apiVersion: v1\nkind: Config\ncurrent-context: " + contextName + "\n" +
		"clusters:\n  - name: " + contextName + "\n    cluster:\n      server: " + serverURL + "\n      insecure-skip-tls-verify: true\n" +
		"contexts:\n  - name: " + contextName + "\n    context:\n      cluster: " + contextName + "\n      user: u\n" +
		"users:\n  - name: u\n    user:\n      token: fake\n"
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

// writeConfig writes a RudderConfig to configDir/config.yaml.
func writeConfig(t *testing.T, configDir string, cfg *config.RudderConfig) {
	t.Helper()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// writeState writes a State to configDir/state.yaml.
func writeState(t *testing.T, configDir, activeEnv string) {
	t.Helper()
	data, err := yaml.Marshal(&config.State{ActiveEnvironment: activeEnv})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "state.yaml"), data, 0600); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

// readState reads and returns state.yaml from configDir.
func readState(t *testing.T, configDir string) *config.State {
	t.Helper()
	state, err := config.LoadState(configDir)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	return state
}

// mockKubectlPath returns the absolute path to the mock kubectl test binary.
func mockKubectlPath(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../internal/kubectl/testdata/mock_kubectl/kubectl")
	if err != nil {
		t.Fatalf("resolve mock kubectl: %v", err)
	}
	return abs
}
