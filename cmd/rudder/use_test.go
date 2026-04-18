package main

import (
	"strings"
	"sync"
	"testing"

	"gitlab.com/dcresp0/rudder/internal/config"
)

func TestUse_Direct(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube},
			{Name: "staging", Kubeconfig: kube},
		},
	})

	r := mustRunCmd(t, dir, "use", "staging")

	if !strings.Contains(r.Stdout, "staging") {
		t.Errorf("expected 'staging' in success message, got: %q", r.Stdout)
	}

	state := readState(t, dir)
	if state == nil {
		t.Fatal("state is nil after use")
	}
	if state.ActiveEnvironment != "staging" {
		t.Errorf("expected state.active_environment='staging', got %q", state.ActiveEnvironment)
	}
}

func TestUse_Direct_NotFound(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})

	r := runCmd(t, dir, "use", "nonexistent")
	if r.Err == nil {
		t.Error("expected error for unknown environment")
	}
}

func TestUse_WritesStateAtomically(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	envs := []config.Environment{
		{Name: "dev", Kubeconfig: kube},
		{Name: "staging", Kubeconfig: kube},
		{Name: "prod", Kubeconfig: kube},
	}
	writeConfig(t, dir, &config.RudderConfig{Version: "1", Environments: envs})

	// Concurrent use commands must not corrupt state
	const goroutines = 10
	var wg sync.WaitGroup
	names := []string{"dev", "staging", "prod"}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			runCmd(t, dir, "use", names[i%len(names)])
		}(i)
	}
	wg.Wait()

	state := readState(t, dir)
	if state == nil {
		t.Fatal("state is nil after concurrent use commands")
	}

	valid := false
	for _, n := range names {
		if state.ActiveEnvironment == n {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("unexpected active environment after concurrent writes: %q", state.ActiveEnvironment)
	}
}

func TestUse_PlainMode_NoEnvs(t *testing.T) {
	dir := t.TempDir()
	r := mustRunCmd(t, dir, "use")
	if !strings.Contains(r.Stdout, "no environments") {
		t.Errorf("expected empty message, got: %q", r.Stdout)
	}
}
