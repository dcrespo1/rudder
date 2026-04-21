package kubeconfig

import (
	"fmt"
	"os"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// MergeKubeconfigs merges multiple kubeconfig files into a single temp file.
// The caller is responsible for removing the returned temp file when done.
func MergeKubeconfigs(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no kubeconfig paths provided")
	}

	merged := clientcmdapi.NewConfig()

	for _, path := range paths {
		cfg, err := clientcmd.LoadFromFile(path)
		if err != nil {
			return "", fmt.Errorf("load kubeconfig %s: %w", path, err)
		}
		for name, cluster := range cfg.Clusters {
			merged.Clusters[name] = cluster
		}
		for name, ctx := range cfg.Contexts {
			merged.Contexts[name] = ctx
		}
		for name, user := range cfg.AuthInfos {
			merged.AuthInfos[name] = user
		}
		if merged.CurrentContext == "" {
			merged.CurrentContext = cfg.CurrentContext
		}
	}

	f, err := os.CreateTemp("", "rudder-kubeconfig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temp kubeconfig: %w", err)
	}
	defer f.Close()

	if err := clientcmd.WriteToFile(*merged, f.Name()); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write merged kubeconfig: %w", err)
	}

	return f.Name(), nil
}
