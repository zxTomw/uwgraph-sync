package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWaterlooBaseURL        = "https://openapi.data.uwaterloo.ca"
	defaultNeo4JURI               = "bolt://localhost:7687"
	defaultNeo4JDatabase          = "neo4j"
	defaultSyncInterval           = 6 * time.Hour
	defaultHTTPTimeout            = 30 * time.Second
	defaultSyncTimeout            = 30 * time.Minute
	defaultStartupTimeout         = 2 * time.Minute
	defaultEmbeddingBaseURL       = "https://api.openai.com/v1"
	defaultEmbeddingBatchSize     = 64
	defaultEmbeddingTimeout       = 30 * time.Second
	defaultEmbeddingPollInterval  = time.Minute
	defaultKnowledgeListenAddress = ":8080"
	defaultKnowledgeQueryTimeout  = 15 * time.Second
)

type Config struct {
	Waterloo       WaterlooConfig
	Neo4J          Neo4JConfig
	TermCodes      []string
	SyncInterval   time.Duration
	SyncTimeout    time.Duration
	StartupTimeout time.Duration
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

type EmbeddingConfig struct {
	BaseURL      string
	APIKey       string
	Model        string
	Dimensions   int
	BatchSize    int
	HTTPTimeout  time.Duration
	PollInterval time.Duration
}

type EmbedConfig struct {
	Neo4J          Neo4JConfig
	Embedding      EmbeddingConfig
	StartupTimeout time.Duration
}

type ServeConfig struct {
	Neo4J             Neo4JConfig
	Embedding         EmbeddingConfig
	StartupTimeout    time.Duration
	ListenAddress     string
	APIKey            string
	MCPAllowedOrigins []string
	QueryTimeout      time.Duration
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

	cfg.Neo4J = loadNeo4J(lookup, &problems)

	cfg.TermCodes = parseTermCodes(required(lookup, "UWGRAPH_TERM_CODES", &problems))
	if len(cfg.TermCodes) == 0 {
		problems = append(problems, errors.New("UWGRAPH_TERM_CODES must contain at least one term code"))
	}

	cfg.SyncInterval = durationOrDefault(lookup, "UWGRAPH_SYNC_INTERVAL", defaultSyncInterval, &problems)
	cfg.SyncTimeout = durationOrDefault(lookup, "UWGRAPH_SYNC_TIMEOUT", defaultSyncTimeout, &problems)
	cfg.StartupTimeout = durationOrDefault(lookup, "UWGRAPH_STARTUP_TIMEOUT", defaultStartupTimeout, &problems)

	if len(problems) > 0 {
		return Config{}, errors.Join(problems...)
	}
	return cfg, nil
}

func LoadEmbed() (EmbedConfig, error) {
	return LoadEmbedFromEnv(os.LookupEnv)
}

func LoadEmbedFromEnv(lookup func(string) (string, bool)) (EmbedConfig, error) {
	var cfg EmbedConfig
	var problems []error

	cfg.Neo4J = loadNeo4J(lookup, &problems)
	cfg.Embedding = loadEmbedding(lookup, &problems)
	cfg.StartupTimeout = durationOrDefault(lookup, "UWGRAPH_STARTUP_TIMEOUT", defaultStartupTimeout, &problems)

	if len(problems) > 0 {
		return EmbedConfig{}, errors.Join(problems...)
	}
	return cfg, nil
}

func LoadServe() (ServeConfig, error) {
	return LoadServeFromEnv(os.LookupEnv)
}

func LoadServeFromEnv(lookup func(string) (string, bool)) (ServeConfig, error) {
	var cfg ServeConfig
	var problems []error

	cfg.Neo4J = loadNeo4J(lookup, &problems)
	cfg.Embedding = loadEmbedding(lookup, &problems)
	cfg.StartupTimeout = durationOrDefault(lookup, "UWGRAPH_STARTUP_TIMEOUT", defaultStartupTimeout, &problems)
	cfg.ListenAddress = valueOrDefault(lookup, "UWGRAPH_KNOWLEDGE_ADDRESS", defaultKnowledgeListenAddress)
	cfg.APIKey = required(lookup, "UWGRAPH_KNOWLEDGE_API_KEY", &problems)
	cfg.MCPAllowedOrigins = parseCSV(valueOrDefault(lookup, "UWGRAPH_MCP_ALLOWED_ORIGINS", ""))
	cfg.QueryTimeout = durationOrDefault(lookup, "UWGRAPH_KNOWLEDGE_QUERY_TIMEOUT", defaultKnowledgeQueryTimeout, &problems)

	if len(problems) > 0 {
		return ServeConfig{}, errors.Join(problems...)
	}
	return cfg, nil
}

func loadNeo4J(lookup func(string) (string, bool), problems *[]error) Neo4JConfig {
	return Neo4JConfig{
		URI:      valueOrDefault(lookup, "NEO4J_URI", defaultNeo4JURI),
		Username: required(lookup, "NEO4J_USERNAME", problems),
		Password: required(lookup, "NEO4J_PASSWORD", problems),
		Database: valueOrDefault(lookup, "NEO4J_DATABASE", defaultNeo4JDatabase),
	}
}

func loadEmbedding(lookup func(string) (string, bool), problems *[]error) EmbeddingConfig {
	return EmbeddingConfig{
		BaseURL:      valueOrDefault(lookup, "UWGRAPH_EMBEDDING_BASE_URL", defaultEmbeddingBaseURL),
		APIKey:       required(lookup, "UWGRAPH_EMBEDDING_API_KEY", problems),
		Model:        required(lookup, "UWGRAPH_EMBEDDING_MODEL", problems),
		Dimensions:   positiveInt(lookup, "UWGRAPH_EMBEDDING_DIMENSIONS", 0, problems),
		BatchSize:    positiveInt(lookup, "UWGRAPH_EMBEDDING_BATCH_SIZE", defaultEmbeddingBatchSize, problems),
		HTTPTimeout:  durationOrDefault(lookup, "UWGRAPH_EMBEDDING_TIMEOUT", defaultEmbeddingTimeout, problems),
		PollInterval: durationOrDefault(lookup, "UWGRAPH_EMBEDDING_POLL_INTERVAL", defaultEmbeddingPollInterval, problems),
	}
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

func positiveInt(lookup func(string) (string, bool), key string, fallback int, problems *[]error) int {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		if fallback <= 0 {
			*problems = append(*problems, fmt.Errorf("%s is required", key))
		}
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		*problems = append(*problems, fmt.Errorf("%s must be an integer: %w", key, err))
		return fallback
	}
	if parsed <= 0 {
		*problems = append(*problems, fmt.Errorf("%s must be greater than zero", key))
		return fallback
	}
	return parsed
}

func parseTermCodes(raw string) []string {
	return parseCSV(raw)
}

func parseCSV(raw string) []string {
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
