package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLoadConfig_NotExist(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("expected no error for missing config, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", cfg.Version)
	}
	if len(cfg.Environments) != 0 {
		t.Errorf("expected empty environments, got %d", len(cfg.Environments))
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	src := filepath.Join("testdata", "config.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Environments) != 3 {
		t.Errorf("expected 3 environments, got %d", len(cfg.Environments))
	}
	if cfg.Environments[0].Name != "dev" {
		t.Errorf("expected first env 'dev', got %q", cfg.Environments[0].Name)
	}
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &RudderConfig{
		Version: "1",
		Environments: []Environment{
			{Name: "test", Kubeconfig: "/tmp/test.yaml", Context: "test-ctx"},
		},
	}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(loaded.Environments) != 1 {
		t.Errorf("expected 1 environment, got %d", len(loaded.Environments))
	}
	if loaded.Environments[0].Name != "test" {
		t.Errorf("expected env name 'test', got %q", loaded.Environments[0].Name)
	}
}

func TestLoadState_NotExist(t *testing.T) {
	dir := t.TempDir()
	state, err := LoadState(dir)
	if err != nil {
		t.Fatalf("expected no error for missing state, got: %v", err)
	}
	if state != nil {
		t.Errorf("expected nil state, got %+v", state)
	}
}

func TestSaveState_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	state := &State{ActiveEnvironment: "staging"}

	if err := SaveState(dir, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// No .rudder-tmp-* files should remain after successful save
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".rudder-tmp-") {
			t.Errorf("temp file still exists after SaveState: %s", e.Name())
		}
	}

	// Final file must exist
	final := filepath.Join(dir, "state.yaml")
	if _, err := os.Stat(final); err != nil {
		t.Errorf("final state file does not exist: %v", err)
	}

	loaded, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded.ActiveEnvironment != "staging" {
		t.Errorf("expected 'staging', got %q", loaded.ActiveEnvironment)
	}
}

func TestSaveState_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	const goroutines = 20
	envs := []string{"dev", "staging", "prod"}

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			env := envs[i%len(envs)]
			_ = SaveState(dir, &State{ActiveEnvironment: env})
		}(i)
	}
	wg.Wait()

	// File must exist and be valid YAML after concurrent writes
	state, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState after concurrent writes: %v", err)
	}
	if state == nil {
		t.Fatal("state is nil after concurrent writes")
	}
	found := false
	for _, e := range envs {
		if state.ActiveEnvironment == e {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected active environment after concurrent writes: %q", state.ActiveEnvironment)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandPath_EnvVar(t *testing.T) {
	t.Setenv("RUDDER_TEST_DIR", "/tmp/testdir")
	got := ExpandPath("$RUDDER_TEST_DIR/kubeconfig.yaml")
	want := "/tmp/testdir/kubeconfig.yaml"
	if got != want {
		t.Errorf("ExpandPath env var: got %q, want %q", got, want)
	}
}
