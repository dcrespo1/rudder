package kubeconfig

import (
	"testing"
)

const fixtureKubeconfig = "testdata/single.yaml"

func TestGetCurrentContext(t *testing.T) {
	ctx, err := GetCurrentContext(fixtureKubeconfig)
	if err != nil {
		t.Fatalf("GetCurrentContext: %v", err)
	}
	if ctx != "kind-dev" {
		t.Errorf("expected 'kind-dev', got %q", ctx)
	}
}

func TestGetCurrentContext_NotExist(t *testing.T) {
	_, err := GetCurrentContext("/nonexistent/kubeconfig.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestListContexts(t *testing.T) {
	ctxs, err := ListContexts(fixtureKubeconfig)
	if err != nil {
		t.Fatalf("ListContexts: %v", err)
	}
	if len(ctxs) != 2 {
		t.Errorf("expected 2 contexts, got %d: %v", len(ctxs), ctxs)
	}
	found := make(map[string]bool)
	for _, c := range ctxs {
		found[c] = true
	}
	for _, want := range []string{"kind-dev", "kind-other"} {
		if !found[want] {
			t.Errorf("expected context %q in list", want)
		}
	}
}

func TestContextExists(t *testing.T) {
	ok, err := ContextExists(fixtureKubeconfig, "kind-dev")
	if err != nil {
		t.Fatalf("ContextExists: %v", err)
	}
	if !ok {
		t.Error("expected 'kind-dev' to exist")
	}

	ok, err = ContextExists(fixtureKubeconfig, "nonexistent")
	if err != nil {
		t.Fatalf("ContextExists: %v", err)
	}
	if ok {
		t.Error("expected 'nonexistent' to not exist")
	}
}
