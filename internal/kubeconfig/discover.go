package kubeconfig

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoverKubeconfigs scans known locations and returns existing kubeconfig paths.
// Checks ~/.kube/config, all *.yaml and *.yml files in ~/.kube/, and paths in
// the KUBECONFIG environment variable. Results are deduplicated.
func DiscoverKubeconfigs() []string {
	seen := make(map[string]struct{})
	var result []string

	add := func(p string) {
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, ok := seen[abs]; ok {
			return
		}
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			return
		}
		seen[abs] = struct{}{}
		result = append(result, abs)
	}

	// ~/.kube/config
	home, err := os.UserHomeDir()
	if err == nil {
		add(filepath.Join(home, ".kube", "config"))

		// All *.yaml and *.yml in ~/.kube/
		entries, err := os.ReadDir(filepath.Join(home, ".kube"))
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
					add(filepath.Join(home, ".kube", name))
				}
			}
		}
	}

	// KUBECONFIG env var (colon-separated paths)
	if env := os.Getenv("KUBECONFIG"); env != "" {
		for _, p := range strings.Split(env, ":") {
			add(strings.TrimSpace(p))
		}
	}

	return result
}
