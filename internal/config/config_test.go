package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnvParsesDefaultsAndTermCodes(t *testing.T) {
	env := map[string]string{
		"WATERLOO_API_KEY":   "waterloo-key",
		"NEO4J_USERNAME":     "neo4j",
		"NEO4J_PASSWORD":     "password",
		"UWGRAPH_TERM_CODES": "1251, 1255,1251",
	}

	cfg, err := LoadFromEnv(mapLookup(env))
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.Waterloo.BaseURL != defaultWaterlooBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.Waterloo.BaseURL, defaultWaterlooBaseURL)
	}
	if cfg.Neo4J.URI != defaultNeo4JURI {
		t.Fatalf("Neo4J URI = %q, want %q", cfg.Neo4J.URI, defaultNeo4JURI)
	}
	if got, want := strings.Join(cfg.TermCodes, ","), "1251,1255"; got != want {
		t.Fatalf("TermCodes = %q, want %q", got, want)
	}
	if cfg.SyncInterval != 6*time.Hour {
		t.Fatalf("SyncInterval = %s, want 6h", cfg.SyncInterval)
	}
	if cfg.StartupTimeout != 2*time.Minute {
		t.Fatalf("StartupTimeout = %s, want 2m", cfg.StartupTimeout)
	}
}

func TestLoadFromEnvRejectsMissingRequiredValues(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{}))
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error")
	}

	for _, want := range []string{"WATERLOO_API_KEY is required", "NEO4J_USERNAME is required", "NEO4J_PASSWORD is required", "UWGRAPH_TERM_CODES is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestLoadFromEnvRejectsInvalidDurations(t *testing.T) {
	env := map[string]string{
		"WATERLOO_API_KEY":        "waterloo-key",
		"NEO4J_USERNAME":          "neo4j",
		"NEO4J_PASSWORD":          "password",
		"UWGRAPH_TERM_CODES":      "1251",
		"UWGRAPH_SYNC_INTERVAL":   "soon",
		"UWGRAPH_SYNC_TIMEOUT":    "0s",
		"UWGRAPH_STARTUP_TIMEOUT": "-1s",
	}

	_, err := LoadFromEnv(mapLookup(env))
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error")
	}
	for _, want := range []string{
		"UWGRAPH_SYNC_INTERVAL must be a valid duration",
		"UWGRAPH_SYNC_TIMEOUT must be greater than zero",
		"UWGRAPH_STARTUP_TIMEOUT must be greater than zero",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
