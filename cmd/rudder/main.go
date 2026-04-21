package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if err := Execute(); err != nil {
		// silentError: subprocess already printed its message; just exit 1.
		var silent *silentError
		if errors.As(err, &silent) {
			// Propagate the subprocess exit code if available.
			var exitErr *exec.ExitError
			if errors.As(silent.cause, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
		// Print non-empty errors to stderr.
		if err.Error() != "" {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
		os.Exit(1)
	}
}
