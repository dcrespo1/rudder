package config

import (
	"fmt"
	"os"
	"regexp"
)

// ValidationError describes a single validation failure on a named field.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

var validName = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ValidateEnvironment returns all validation errors for a single environment.
func ValidateEnvironment(env *Environment) []ValidationError {
	var errs []ValidationError

	if env.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "must not be empty"})
	} else if !validName.MatchString(env.Name) {
		errs = append(errs, ValidationError{Field: "name", Message: "must contain only lowercase letters, digits, hyphens, and underscores"})
	}

	expanded := ExpandPath(env.Kubeconfig)
	if expanded == "" {
		errs = append(errs, ValidationError{Field: "kubeconfig", Message: "must not be empty"})
	} else {
		info, err := os.Stat(expanded)
		if err != nil {
			errs = append(errs, ValidationError{Field: "kubeconfig", Message: fmt.Sprintf("path does not exist: %s", expanded)})
		} else if info.IsDir() {
			errs = append(errs, ValidationError{Field: "kubeconfig", Message: fmt.Sprintf("path is a directory, not a file: %s", expanded)})
		}
	}

	return errs
}

// ValidateConfig returns all validation errors across all environments.
func ValidateConfig(cfg *RudderConfig) []ValidationError {
	var errs []ValidationError
	for _, env := range cfg.Environments {
		env := env
		for _, e := range ValidateEnvironment(&env) {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("%s.%s", env.Name, e.Field),
				Message: e.Message,
			})
		}
	}
	return errs
}
