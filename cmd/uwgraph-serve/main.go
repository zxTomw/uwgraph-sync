package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"uwgraph/internal/app"
	"uwgraph/internal/config"
	"uwgraph/internal/embedding"
	"uwgraph/internal/knowledgeapi"
	"uwgraph/internal/neo4jstore"
	"uwgraph/internal/retrieval"
)

func main() {
	logger := app.Logger()
	app.LoadDotEnv(logger)
	cfg, err := config.LoadServe()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := app.SignalContext()
	defer stop()

	driver, err := app.OpenNeo4J(ctx, cfg.Neo4J, cfg.StartupTimeout, logger)
	if err != nil {
		logger.Error("connect to neo4j", "error", err)
		os.Exit(1)
	}
	defer app.CloseNeo4J(driver, logger)

	store := neo4jstore.New(driver, cfg.Neo4J.Database, logger)
	provider := embedding.NewOpenAICompatibleClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.Model,
		cfg.Embedding.Dimensions,
		cfg.Embedding.HTTPTimeout,
	)
	service := retrieval.NewService(store, provider)
	readyCtx, readyCancel := context.WithTimeout(ctx, cfg.QueryTimeout)
	if err := service.Ready(readyCtx); err != nil {
		readyCancel()
		logger.Error("knowledge service is not ready", "error", err)
		os.Exit(1)
	}
	readyCancel()

	api := knowledgeapi.New(
		service,
		cfg.APIKey,
		cfg.MCPAllowedOrigins,
		cfg.QueryTimeout,
		logger,
	)
	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("knowledge service listening", "address", cfg.ListenAddress)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown knowledge service", "error", err)
			os.Exit(1)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("knowledge service stopped", "error", err)
			os.Exit(1)
		}
	}
}
