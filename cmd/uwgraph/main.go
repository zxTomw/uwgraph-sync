package main

import (
	"os"

	"uwgraph/internal/app"
	"uwgraph/internal/config"
	"uwgraph/internal/neo4jstore"
	"uwgraph/internal/runner"
	"uwgraph/internal/syncer"
	"uwgraph/internal/waterloo"
)

func main() {
	logger := app.Logger()
	app.LoadDotEnv(logger)

	cfg, err := config.Load()
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

	waterlooClient := waterloo.NewClient(cfg.Waterloo.BaseURL, cfg.Waterloo.APIKey, cfg.Waterloo.HTTPTimeout, logger)
	store := neo4jstore.New(driver, cfg.Neo4J.Database, logger)
	syncService := syncer.New(waterlooClient, store, cfg.TermCodes, logger)
	serviceRunner := runner.New(syncService.Sync, cfg.SyncInterval, cfg.SyncTimeout, logger)

	if err := serviceRunner.Run(ctx); err != nil {
		logger.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}
