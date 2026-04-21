package config

// RudderConfig is the top-level schema for ~/.rudder/config.yaml.
type RudderConfig struct {
	Version          string        `yaml:"version"`
	KubectlPath      string        `yaml:"kubectl_path,omitempty"`
	DefaultNamespace string        `yaml:"default_namespace,omitempty"`
	Environments     []Environment `yaml:"environments"`
}

// Environment maps a named alias to a Kubernetes cluster context.
type Environment struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Kubeconfig  string   `yaml:"kubeconfig"`
	Context     string   `yaml:"context,omitempty"`
	Namespace   string   `yaml:"namespace,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// State is the schema for ~/.rudder/state.yaml.
type State struct {
	ActiveEnvironment string `yaml:"active_environment"`
}

// FindEnv returns the environment with the given name and true, or nil and false.
func (c *RudderConfig) FindEnv(name string) (*Environment, bool) {
	for i := range c.Environments {
		if c.Environments[i].Name == name {
			return &c.Environments[i], true
		}
	}
	return nil, false
}

// FilterByTags returns environments that have all of the listed tags.
func (c *RudderConfig) FilterByTags(tags []string) []Environment {
	if len(tags) == 0 {
		return c.Environments
	}
	var result []Environment
	for _, env := range c.Environments {
		if envHasAllTags(env, tags) {
			result = append(result, env)
		}
	}
	return result
}

func envHasAllTags(env Environment, required []string) bool {
	tagSet := make(map[string]struct{}, len(env.Tags))
	for _, t := range env.Tags {
		tagSet[t] = struct{}{}
	}
	for _, r := range required {
		if _, ok := tagSet[r]; !ok {
			return false
		}
	}
	return true
}
