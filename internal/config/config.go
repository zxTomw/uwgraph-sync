package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultWaterlooBaseURL = "https://openapi.data.uwaterloo.ca"
	defaultNeo4JURI        = "bolt://localhost:7687"
	defaultNeo4JDatabase   = "neo4j"
	defaultSyncInterval    = 6 * time.Hour
	defaultHTTPTimeout     = 30 * time.Second
	defaultSyncTimeout     = 30 * time.Minute
)

type Config struct {
	Waterloo     WaterlooConfig
	Neo4J        Neo4JConfig
	TermCodes    []string
	SyncInterval time.Duration
	SyncTimeout  time.Duration
}

type WaterlooConfig struct {
	APIKey      string
	BaseURL     string
	HTTPTimeout time.Duration
}

type Neo4JConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

func LoadFromEnv(lookup func(string) (string, bool)) (Config, error) {
	var cfg Config
	var problems []error

	cfg.Waterloo.APIKey = required(lookup, "WATERLOO_API_KEY", &problems)
	cfg.Waterloo.BaseURL = valueOrDefault(lookup, "WATERLOO_BASE_URL", defaultWaterlooBaseURL)
	cfg.Waterloo.HTTPTimeout = durationOrDefault(lookup, "UWGRAPH_HTTP_TIMEOUT", defaultHTTPTimeout, &problems)

	cfg.Neo4J.URI = valueOrDefault(lookup, "NEO4J_URI", defaultNeo4JURI)
	cfg.Neo4J.Username = required(lookup, "NEO4J_USERNAME", &problems)
	cfg.Neo4J.Password = required(lookup, "NEO4J_PASSWORD", &problems)
	cfg.Neo4J.Database = valueOrDefault(lookup, "NEO4J_DATABASE", defaultNeo4JDatabase)

	cfg.TermCodes = parseTermCodes(required(lookup, "UWGRAPH_TERM_CODES", &problems))
	if len(cfg.TermCodes) == 0 {
		problems = append(problems, errors.New("UWGRAPH_TERM_CODES must contain at least one term code"))
	}

	cfg.SyncInterval = durationOrDefault(lookup, "UWGRAPH_SYNC_INTERVAL", defaultSyncInterval, &problems)
	cfg.SyncTimeout = durationOrDefault(lookup, "UWGRAPH_SYNC_TIMEOUT", defaultSyncTimeout, &problems)

	if len(problems) > 0 {
		return Config{}, errors.Join(problems...)
	}
	return cfg, nil
}

func required(lookup func(string) (string, bool), key string, problems *[]error) string {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		*problems = append(*problems, fmt.Errorf("%s is required", key))
	}
	return value
}

func valueOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func durationOrDefault(lookup func(string) (string, bool), key string, fallback time.Duration, problems *[]error) time.Duration {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		*problems = append(*problems, fmt.Errorf("%s must be a valid duration: %w", key, err))
		return fallback
	}
	if duration <= 0 {
		*problems = append(*problems, fmt.Errorf("%s must be greater than zero", key))
		return fallback
	}
	return duration
}

func parseTermCodes(raw string) []string {
	parts := strings.Split(raw, ",")
	codes := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes
}
