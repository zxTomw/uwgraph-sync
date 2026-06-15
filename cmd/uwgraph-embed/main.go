package main

import (
	"flag"
	"os"

	"uwgraph/internal/app"
	"uwgraph/internal/config"
	"uwgraph/internal/embedding"
	"uwgraph/internal/neo4jstore"
)

func main() {
	once := flag.Bool("once", false, "embed all stale knowledge documents and exit")
	rebuildIndex := flag.Bool("rebuild-index", false, "recreate the vector index and mark all documents stale")
	flag.Parse()

	logger := app.Logger()
	app.LoadDotEnv(logger)
	cfg, err := config.LoadEmbed()
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
	if err := store.EnsureSchema(ctx); err != nil {
		logger.Error("ensure knowledge schema", "error", err)
		os.Exit(1)
	}
	if *rebuildIndex {
		if err := store.RebuildVectorIndex(ctx, cfg.Embedding.Dimensions); err != nil {
			logger.Error("rebuild vector index", "error", err)
			os.Exit(1)
		}
	}
	provider := embedding.NewOpenAICompatibleClient(
		cfg.Embedding.BaseURL,
		cfg.Embedding.APIKey,
		cfg.Embedding.Model,
		cfg.Embedding.Dimensions,
		cfg.Embedding.HTTPTimeout,
	)
	worker := embedding.NewWorker(
		store,
		provider,
		cfg.Embedding.Model,
		cfg.Embedding.Dimensions,
		cfg.Embedding.BatchSize,
		cfg.Embedding.PollInterval,
		logger,
	)
	if *once {
		count, err := worker.RunOnce(ctx)
		if err != nil {
			logger.Error("embed knowledge documents", "error", err)
			os.Exit(1)
		}
		logger.Info("embedding backfill completed", "updated", count)
		return
	}
	if err := worker.Run(ctx); err != nil {
		logger.Error("embedding worker stopped", "error", err)
		os.Exit(1)
	}
}
