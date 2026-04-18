package kubeconfig

import (
	"os"
	"testing"
)

func TestMergeKubeconfigs(t *testing.T) {
	paths := []string{"testdata/single.yaml", "testdata/second.yaml"}
	merged, err := MergeKubeconfigs(paths)
	if err != nil {
		t.Fatalf("MergeKubeconfigs: %v", err)
	}
	defer os.Remove(merged)

	if _, err := os.Stat(merged); err != nil {
		t.Fatalf("merged file does not exist: %v", err)
	}

	// Merged file should contain contexts from both sources
	ctxs, err := ListContexts(merged)
	if err != nil {
		t.Fatalf("ListContexts on merged: %v", err)
	}
	found := make(map[string]bool)
	for _, c := range ctxs {
		found[c] = true
	}
	for _, want := range []string{"kind-dev", "kind-other", "aks-staging"} {
		if !found[want] {
			t.Errorf("expected context %q in merged file, got contexts: %v", want, ctxs)
		}
	}
}

func TestMergeKubeconfigs_Empty(t *testing.T) {
	_, err := MergeKubeconfigs(nil)
	if err == nil {
		t.Error("expected error for empty paths")
	}
}

func TestMergeKubeconfigs_NotExist(t *testing.T) {
	_, err := MergeKubeconfigs([]string{"/nonexistent/kubeconfig.yaml"})
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}
