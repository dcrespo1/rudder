package main

import (
	"encoding/json"
	"strings"
	"testing"

	"gitlab.com/dcresp0/rudder/internal/config"
)

func TestEnvs_Empty(t *testing.T) {
	dir := t.TempDir()
	r := mustRunCmd(t, dir, "envs")
	if !strings.Contains(r.Stdout, "no environments") {
		t.Errorf("expected empty message, got: %q", r.Stdout)
	}
}

func TestEnvs_Table(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "dev", "https://127.0.0.1:6443", "kind-dev")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube, Context: "kind-dev", Namespace: "default", Tags: []string{"local"}},
			{Name: "prod", Kubeconfig: kube, Context: "aks-prod", Tags: []string{"azure"}},
		},
	})
	writeState(t, dir, "dev")

	r := mustRunCmd(t, dir, "envs")

	if !strings.Contains(r.Stdout, "dev") {
		t.Errorf("expected 'dev' in output, got: %q", r.Stdout)
	}
	if !strings.Contains(r.Stdout, "prod") {
		t.Errorf("expected 'prod' in output, got: %q", r.Stdout)
	}
	// Active marker for dev
	if !strings.Contains(r.Stdout, "●") {
		t.Errorf("expected active marker ● in output, got: %q", r.Stdout)
	}
}

func TestEnvs_JSON(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "dev", "https://127.0.0.1:6443", "kind-dev")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube, Tags: []string{"local"}},
			{Name: "prod", Kubeconfig: kube, Tags: []string{"azure", "prod"}},
		},
	})
	writeState(t, dir, "dev")

	r := mustRunCmd(t, dir, "envs", "--output", "json")

	var out []envJSON
	if err := json.Unmarshal([]byte(r.Stdout), &out); err != nil {
		t.Fatalf("parse JSON: %v\noutput: %s", err, r.Stdout)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 envs, got %d", len(out))
	}

	devFound := false
	for _, e := range out {
		if e.Name == "dev" {
			devFound = true
			if !e.Active {
				t.Error("expected dev to be marked active")
			}
		}
	}
	if !devFound {
		t.Error("expected 'dev' in JSON output")
	}
}

func TestEnvs_TagFilter(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube, Tags: []string{"local"}},
			{Name: "staging", Kubeconfig: kube, Tags: []string{"azure", "aks"}},
			{Name: "prod", Kubeconfig: kube, Tags: []string{"azure", "aks", "prod"}},
		},
	})

	r := mustRunCmd(t, dir, "envs", "--tags", "azure")

	if strings.Contains(r.Stdout, "dev") {
		t.Error("'dev' should be filtered out")
	}
	if !strings.Contains(r.Stdout, "staging") {
		t.Error("expected 'staging' in filtered output")
	}
	if !strings.Contains(r.Stdout, "prod") {
		t.Error("expected 'prod' in filtered output")
	}
}

func TestEnvs_RudderEnvOverride(t *testing.T) {
	dir := t.TempDir()
	kube := writeKubeconfig(t, dir, "k", "https://127.0.0.1:6443", "ctx")

	writeConfig(t, dir, &config.RudderConfig{
		Version: "1",
		Environments: []config.Environment{
			{Name: "dev", Kubeconfig: kube},
			{Name: "prod", Kubeconfig: kube},
		},
	})
	writeState(t, dir, "dev") // persisted active = dev

	t.Setenv("RUDDER_ENV", "prod") // override to prod

	r := mustRunCmd(t, dir, "envs", "-o", "json")

	var out []envJSON
	if err := json.Unmarshal([]byte(r.Stdout), &out); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	for _, e := range out {
		if e.Name == "prod" && !e.Active {
			t.Error("expected 'prod' to be active via RUDDER_ENV override")
		}
		if e.Name == "dev" && e.Active {
			t.Error("'dev' should not be active when RUDDER_ENV=prod")
		}
	}
}
