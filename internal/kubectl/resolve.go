package kubectl

import (
	"fmt"
	"os"
	"os/exec"
)

// Resolve returns the path to the kubectl binary.
// Resolution order: overridePath → RUDDER_KUBECTL env → PATH lookup.
func Resolve(overridePath string) (string, error) {
	if overridePath != "" {
		if err := checkExecutable(overridePath); err != nil {
			return "", fmt.Errorf("kubectl override path %q: %w", overridePath, err)
		}
		return overridePath, nil
	}

	if env := os.Getenv("RUDDER_KUBECTL"); env != "" {
		if err := checkExecutable(env); err != nil {
			return "", fmt.Errorf("RUDDER_KUBECTL=%q: %w", env, err)
		}
		return env, nil
	}

	path, err := exec.LookPath("kubectl")
	if err != nil {
		return "", fmt.Errorf("kubectl not found on PATH — install kubectl or set RUDDER_KUBECTL: %w", err)
	}
	return path, nil
}

func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable")
	}
	return nil
}
