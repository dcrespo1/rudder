package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"gitlab.com/dcresp0/rudder/internal/config"
)

// Status represents the reachability state of a cluster.
type Status int

const (
	StatusUnknown    Status = iota
	StatusProbing           // in-flight check
	StatusReachable         // HTTP 200 on /readyz
	StatusUnreachable       // connection failed or non-200
)

// PingResult carries the result of a single probe.
type PingResult struct {
	EnvName string
	Status  Status
}

// Ping performs a lightweight reachability check against a cluster's /readyz endpoint.
// It respects ctx cancellation. A short-lived http.Client with a 2s timeout is always
// used — never http.DefaultClient.
func Ping(ctx context.Context, kubeconfigPath, contextName string) Status {
	serverURL, tlsCfg, err := extractServerInfo(kubeconfigPath, contextName)
	if err != nil {
		return StatusUnreachable
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/readyz", nil)
	if err != nil {
		return StatusUnreachable
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnreachable
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusReachable
	}
	return StatusUnreachable
}

// PingAll concurrently pings all environments and streams results back on the returned channel.
// The channel is closed once all probes complete or ctx is cancelled.
func PingAll(ctx context.Context, envs []config.Environment) <-chan PingResult {
	ch := make(chan PingResult, len(envs))

	go func() {
		defer close(ch)

		done := make(chan struct{})
		pending := len(envs)
		results := make(chan PingResult, len(envs))

		for _, env := range envs {
			env := env
			go func() {
				probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				kubeconfigPath := config.ExpandPath(env.Kubeconfig)
				status := Ping(probeCtx, kubeconfigPath, env.Context)
				results <- PingResult{EnvName: env.Name, Status: status}
			}()
		}

		go func() {
			for i := 0; i < pending; i++ {
				select {
				case r := <-results:
					ch <- r
				case <-ctx.Done():
					return
				}
			}
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
		}
	}()

	return ch
}

// extractServerInfo loads a kubeconfig and returns the API server URL and TLS config
// for the given context. Uses the kubeconfig's CA cert and optional client cert/key.
func extractServerInfo(kubeconfigPath, contextName string) (string, *tls.Config, error) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	// Resolve which context to use
	ctxName := contextName
	if ctxName == "" {
		ctxName = cfg.CurrentContext
	}
	ctx, ok := cfg.Contexts[ctxName]
	if !ok {
		return "", nil, fmt.Errorf("context %q not found in kubeconfig", ctxName)
	}

	cluster, ok := cfg.Clusters[ctx.Cluster]
	if !ok {
		return "", nil, fmt.Errorf("cluster %q not found in kubeconfig", ctx.Cluster)
	}
	if cluster.Server == "" {
		return "", nil, fmt.Errorf("cluster %q has no server URL", ctx.Cluster)
	}

	tlsCfg := &tls.Config{} //nolint:gosec

	if cluster.InsecureSkipTLSVerify {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec
	} else if len(cluster.CertificateAuthorityData) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(cluster.CertificateAuthorityData)
		tlsCfg.RootCAs = pool
	}

	// Client cert/key for mTLS
	authInfo, ok := cfg.AuthInfos[ctx.AuthInfo]
	if ok && len(authInfo.ClientCertificateData) > 0 && len(authInfo.ClientKeyData) > 0 {
		cert, err := tls.X509KeyPair(authInfo.ClientCertificateData, authInfo.ClientKeyData)
		if err == nil {
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
	}

	return cluster.Server, tlsCfg, nil
}
