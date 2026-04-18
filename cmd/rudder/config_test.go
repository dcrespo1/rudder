package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/dcresp0/rudder/internal/config"
)

func TestConfigAdd_Flags(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "dev", "https://127.0.0.1:6443", "kind-dev")

	r := mustRunCmd(t, dir, "config", "add",
		"--name", "dev",
		"--kubeconfig", kube,
		"--context", "kind-dev",
		"--namespace", "default",
		"--tags", "local,kind",
		"--description", "Local dev cluster",
	)

	if !strings.Contains(r.Stdout, "added") {
		t.Errorf("expected success message, got: %q", r.Stdout)
	}

	cfg, err := config.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	env, ok := cfg.FindEnv("dev")
	if !ok {
		t.Fatal("expected 'dev' env in config")
	}
	if env.Context != "kind-dev" {
		t.Errorf("expected context 'kind-dev', got %q", env.Context)
	}
	if env.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", env.Namespace)
	}
	if len(env.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(env.Tags), env.Tags)
	}
}

func TestConfigAdd_DuplicateName(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	mustRunCmd(t, dir, "config", "add", "--name", "dev", "--kubeconfig", kube)

	r := runCmd(t, dir, "config", "add", "--name", "dev", "--kubeconfig", kube)
	if r.Err == nil {
		t.Error("expected error for duplicate environment name")
	}
}

func TestConfigAdd_InvalidKubeconfig(t *testing.T) {
	dir := t.TempDir()

	r := runCmd(t, dir, "config", "add",
		"--name", "dev",
		"--kubeconfig", "/nonexistent/path.yaml",
	)
	if r.Err == nil {
		t.Error("expected validation error for nonexistent kubeconfig")
	}
}

func TestConfigRemove(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube},
			{Name: "staging", Kubeconfig: kube},
		},
	})

	mustRunCmd(t, dir, "config", "remove", "staging", "--yes")

	cfg, _ := config.LoadConfig(dir)
	if _, ok := cfg.FindEnv("staging"); ok {
		t.Error("expected 'staging' to be removed")
	}
	if _, ok := cfg.FindEnv("dev"); !ok {
		t.Error("'dev' should still exist")
	}
}

func TestConfigRemove_ClearsActiveState(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})
	writeState(t, dir, "dev")

	mustRunCmd(t, dir, "config", "remove", "dev", "--yes")

	state := readState(t, dir)
	if state != nil && state.ActiveEnvironment == "dev" {
		t.Error("expected state to be cleared after removing active environment")
	}
}

func TestConfigRemove_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := runCmd(t, dir, "config", "remove", "nonexistent", "--yes")
	if r.Err == nil {
		t.Error("expected error removing nonexistent environment")
	}
}

func TestConfigRename(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})

	mustRunCmd(t, dir, "config", "rename", "dev", "development")

	cfg, _ := config.LoadConfig(dir)
	if _, ok := cfg.FindEnv("dev"); ok {
		t.Error("old name 'dev' should not exist after rename")
	}
	if _, ok := cfg.FindEnv("development"); !ok {
		t.Error("new name 'development' should exist after rename")
	}
}

func TestConfigRename_UpdatesActiveState(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})
	writeState(t, dir, "dev")

	mustRunCmd(t, dir, "config", "rename", "dev", "development")

	state := readState(t, dir)
	if state == nil || state.ActiveEnvironment != "development" {
		t.Errorf("expected state to be updated to 'development', got %v", state)
	}
}

func TestConfigRename_DuplicateTarget(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube},
			{Name: "prod", Kubeconfig: kube},
		},
	})

	r := runCmd(t, dir, "config", "rename", "dev", "prod")
	if r.Err == nil {
		t.Error("expected error when renaming to existing name")
	}
}

func TestConfigValidate_AllValid(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version:      "1",
		Environments: []config.Environment{{Name: "dev", Kubeconfig: kube}},
	})

	r := mustRunCmd(t, dir, "config", "validate")
	if !strings.Contains(r.Stdout, "dev") {
		t.Errorf("expected 'dev' in validate output, got: %q", r.Stdout)
	}
}

func TestConfigValidate_InvalidKubeconfig(t *testing.T) {
	dir := t.TempDir()

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "bad", Kubeconfig: "/nonexistent/kubeconfig.yaml"},
		},
	})

	r := runCmd(t, dir, "config", "validate")
	if r.Err == nil {
		t.Error("expected error from validate with invalid kubeconfig")
	}
}

func TestConfigList(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube},
			{Name: "prod", Kubeconfig: kube},
		},
	})

	r := mustRunCmd(t, dir, "config", "list")
	if !strings.Contains(r.Stdout, "dev") || !strings.Contains(r.Stdout, "prod") {
		t.Errorf("expected both envs in config list, got: %q", r.Stdout)
	}
}

func TestConfigEdit_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	// Use a no-op editor
	editorScript := filepath.Join(dir, "editor.sh")
	if err := os.WriteFile(editorScript, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("EDITOR", editorScript)

	mustRunCmd(t, dir, "config", "edit")

	// Config file should have been created
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Errorf("expected config.yaml to be created: %v", err)
	}
}
