package config

import (
	"testing"
)

func TestFindEnv(t *testing.T) {
	cfg := &RudderConfig{
		Environments: []Environment{
			{Name: "dev"},
			{Name: "staging"},
			{Name: "prod"},
		},
	}

	env, ok := cfg.FindEnv("staging")
	if !ok {
		t.Fatal("expected to find 'staging'")
	}
	if env.Name != "staging" {
		t.Errorf("expected 'staging', got %q", env.Name)
	}

	_, ok = cfg.FindEnv("missing")
	if ok {
		t.Error("expected not found for 'missing'")
	}
}

func TestFilterByTags_NoFilter(t *testing.T) {
	cfg := &RudderConfig{
		Environments: []Environment{
			{Name: "dev", Tags: []string{"local"}},
			{Name: "staging", Tags: []string{"azure", "aks"}},
		},
	}
	result := cfg.FilterByTags(nil)
	if len(result) != 2 {
		t.Errorf("expected 2 results with no tag filter, got %d", len(result))
	}
}

func TestFilterByTags_MatchAll(t *testing.T) {
	cfg := &RudderConfig{
		Environments: []Environment{
			{Name: "dev", Tags: []string{"local", "kind"}},
			{Name: "staging", Tags: []string{"azure", "aks", "govcloud"}},
			{Name: "prod", Tags: []string{"azure", "aks", "govcloud", "prod"}},
		},
	}

	result := cfg.FilterByTags([]string{"azure", "govcloud"})
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}

	result = cfg.FilterByTags([]string{"prod"})
	if len(result) != 1 || result[0].Name != "prod" {
		t.Errorf("expected only 'prod', got %v", result)
	}

	result = cfg.FilterByTags([]string{"local", "azure"})
	if len(result) != 0 {
		t.Errorf("expected 0 results for conflicting tags, got %d", len(result))
	}
}
