package kubectl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_Override(t *testing.T) {
	mock := filepath.Join("testdata", "mock_kubectl", "kubectl")
	path, err := Resolve(mock)
	if err != nil {
		t.Fatalf("Resolve with override: %v", err)
	}
	if path != mock {
		t.Errorf("expected %q, got %q", mock, path)
	}
}

func TestResolve_OverrideNotExist(t *testing.T) {
	_, err := Resolve("/nonexistent/kubectl")
	if err == nil {
		t.Error("expected error for nonexistent override path")
	}
}

func TestResolve_EnvVar(t *testing.T) {
	mock, _ := filepath.Abs(filepath.Join("testdata", "mock_kubectl", "kubectl"))
	t.Setenv("RUDDER_KUBECTL", mock)

	path, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve via RUDDER_KUBECTL: %v", err)
	}
	if path != mock {
		t.Errorf("expected %q, got %q", mock, path)
	}
}

func TestResolve_OverrideTakesPrecedenceOverEnv(t *testing.T) {
	mock, _ := filepath.Abs(filepath.Join("testdata", "mock_kubectl", "kubectl"))
	t.Setenv("RUDDER_KUBECTL", "/nonexistent/kubectl")

	path, err := Resolve(mock)
	if err != nil {
		t.Fatalf("Resolve with override: %v", err)
	}
	if path != mock {
		t.Errorf("expected override %q, got %q", mock, path)
	}
}

func TestCheckExecutable_Directory(t *testing.T) {
	dir := t.TempDir()
	err := checkExecutable(dir)
	if err == nil {
		t.Error("expected error for directory")
	}
}

func TestCheckExecutable_NotExecutable(t *testing.T) {
	f, err := os.CreateTemp("", "notexec-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()
	os.Chmod(f.Name(), 0600) // readable but not executable

	err = checkExecutable(f.Name())
	if err == nil {
		t.Error("expected error for non-executable file")
	}
}
