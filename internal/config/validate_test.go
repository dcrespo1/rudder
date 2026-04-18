package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEnvironment_Valid(t *testing.T) {
	// Create a real kubeconfig file so the path check passes
	f, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	env := &Environment{
		Name:       "dev",
		Kubeconfig: f.Name(),
	}
	errs := ValidateEnvironment(env)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateEnvironment_MissingName(t *testing.T) {
	f, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	env := &Environment{Name: "", Kubeconfig: f.Name()}
	errs := ValidateEnvironment(env)
	if len(errs) == 0 {
		t.Error("expected error for empty name")
	}
	if errs[0].Field != "name" {
		t.Errorf("expected field 'name', got %q", errs[0].Field)
	}
}

func TestValidateEnvironment_InvalidNameChars(t *testing.T) {
	f, _ := os.CreateTemp("", "kubeconfig-*.yaml")
	defer os.Remove(f.Name())
	f.Close()

	tests := []struct {
		name  string
		valid bool
	}{
		{"dev", true},
		{"my-cluster", true},
		{"cluster_01", true},
		{"MyCluster", false},
		{"cluster name", false},
		{"cluster.name", false},
	}

	for _, tt := range tests {
		env := &Environment{Name: tt.name, Kubeconfig: f.Name()}
		errs := ValidateEnvironment(env)
		nameErrs := 0
		for _, e := range errs {
			if e.Field == "name" {
				nameErrs++
			}
		}
		if tt.valid && nameErrs > 0 {
			t.Errorf("name %q should be valid but got name error", tt.name)
		}
		if !tt.valid && nameErrs == 0 {
			t.Errorf("name %q should be invalid but got no name error", tt.name)
		}
	}
}

func TestValidateEnvironment_KubeconfigNotExist(t *testing.T) {
	env := &Environment{
		Name:       "dev",
		Kubeconfig: "/nonexistent/path/kubeconfig.yaml",
	}
	errs := ValidateEnvironment(env)
	found := false
	for _, e := range errs {
		if e.Field == "kubeconfig" {
			found = true
		}
	}
	if !found {
		t.Error("expected kubeconfig error for nonexistent path")
	}
}

func TestValidateEnvironment_KubeconfigIsDir(t *testing.T) {
	dir := t.TempDir()
	env := &Environment{
		Name:       "dev",
		Kubeconfig: dir,
	}
	errs := ValidateEnvironment(env)
	found := false
	for _, e := range errs {
		if e.Field == "kubeconfig" {
			found = true
		}
	}
	if !found {
		t.Error("expected kubeconfig error when path is a directory")
	}
}

func TestValidateEnvironment_TildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	// Create a real file under home so expansion + stat passes
	f, err := os.CreateTemp(home, "rudder-test-*.yaml")
	if err != nil {
		t.Fatalf("create temp file in home: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	rel := "~/" + filepath.Base(f.Name())
	env := &Environment{Name: "dev", Kubeconfig: rel}
	errs := ValidateEnvironment(env)
	for _, e := range errs {
		if e.Field == "kubeconfig" {
			t.Errorf("unexpected kubeconfig error with tilde path: %v", e)
		}
	}
}

func TestValidateConfig_MultipleEnvs(t *testing.T) {
	// One valid env (real file), one invalid env (missing kubeconfig)
	f, _ := os.CreateTemp("", "kubeconfig-*.yaml")
	defer os.Remove(f.Name())
	f.Close()

	cfg := &RudderConfig{
		Environments: []Environment{
			{Name: "good", Kubeconfig: f.Name()},
			{Name: "bad", Kubeconfig: "/does/not/exist.yaml"},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Error("expected at least one error from bad env")
	}
	// Error field should be prefixed with env name
	for _, e := range errs {
		if e.Field[:3] != "bad" {
			t.Errorf("expected error field to start with 'bad', got %q", e.Field)
		}
	}
}
