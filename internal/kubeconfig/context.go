package kubeconfig

import (
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

// GetCurrentContext returns the current-context field from a kubeconfig file.
func GetCurrentContext(kubeconfigPath string) (string, error) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("load kubeconfig %s: %w", kubeconfigPath, err)
	}
	if cfg.CurrentContext == "" {
		return "", fmt.Errorf("kubeconfig %s has no current-context set", kubeconfigPath)
	}
	return cfg.CurrentContext, nil
}

// ListContexts returns all context names defined in a kubeconfig file.
func ListContexts(kubeconfigPath string) ([]string, error) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig %s: %w", kubeconfigPath, err)
	}
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	return names, nil
}

// ContextExists reports whether a named context exists in a kubeconfig file.
func ContextExists(kubeconfigPath, contextName string) (bool, error) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return false, fmt.Errorf("load kubeconfig %s: %w", kubeconfigPath, err)
	}
	_, ok := cfg.Contexts[contextName]
	return ok, nil
}
