package cluster

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gitlab.com/dcresp0/rudder/internal/config"
)

// makeKubeconfig writes a minimal kubeconfig pointing at the given server URL.
func makeKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: test-ctx
clusters:
  - name: test-cluster
    cluster:
      server: %s
      insecure-skip-tls-verify: true
contexts:
  - name: test-ctx
    context:
      cluster: test-cluster
      user: test-user
users:
  - name: test-user
    user:
      token: fake-token
`, serverURL)

	f, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("create temp kubeconfig: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestPing_Reachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/readyz" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	kubeconfig := makeKubeconfig(t, srv.URL)
	status := Ping(context.Background(), kubeconfig, "test-ctx")
	if status != StatusReachable {
		t.Errorf("expected StatusReachable, got %v", status)
	}
}

func TestPing_Unreachable_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	kubeconfig := makeKubeconfig(t, srv.URL)
	status := Ping(context.Background(), kubeconfig, "test-ctx")
	if status != StatusUnreachable {
		t.Errorf("expected StatusUnreachable, got %v", status)
	}
}

func TestPing_Unreachable_ConnRefused(t *testing.T) {
	// Use an address that will refuse connections
	kubeconfig := makeKubeconfig(t, "http://127.0.0.1:1")
	status := Ping(context.Background(), kubeconfig, "test-ctx")
	if status != StatusUnreachable {
		t.Errorf("expected StatusUnreachable for refused connection, got %v", status)
	}
}

func TestPing_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client disconnects or a long timeout elapses.
		// Using r.Context() ensures the handler exits quickly when the client cancels.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(10 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	kubeconfig := makeKubeconfig(t, srv.URL)

	// Use a very short context to simulate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	status := Ping(ctx, kubeconfig, "test-ctx")
	if status != StatusUnreachable {
		t.Errorf("expected StatusUnreachable on timeout, got %v", status)
	}
}

func TestPing_BadKubeconfig(t *testing.T) {
	status := Ping(context.Background(), "/nonexistent/kubeconfig.yaml", "ctx")
	if status != StatusUnreachable {
		t.Errorf("expected StatusUnreachable for bad kubeconfig, got %v", status)
	}
}

func TestPingAll(t *testing.T) {
	srv200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv200.Close()

	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()

	kube200 := makeKubeconfig(t, srv200.URL)
	kube500 := makeKubeconfig(t, srv500.URL)

	envs := []config.Environment{
		{Name: "reachable", Kubeconfig: kube200, Context: "test-ctx"},
		{Name: "unreachable", Kubeconfig: kube500, Context: "test-ctx"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := make(map[string]Status)
	for r := range PingAll(ctx, envs) {
		results[r.EnvName] = r.Status
	}

	if results["reachable"] != StatusReachable {
		t.Errorf("expected 'reachable' to be StatusReachable, got %v", results["reachable"])
	}
	if results["unreachable"] != StatusUnreachable {
		t.Errorf("expected 'unreachable' to be StatusUnreachable, got %v", results["unreachable"])
	}
}
