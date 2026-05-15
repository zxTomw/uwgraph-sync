package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/config"
	"uwgraph/internal/neo4jstore"
	"uwgraph/internal/runner"
	"uwgraph/internal/syncer"
	"uwgraph/internal/waterloo"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Warn("load .env", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4J.URI,
		neo4j.BasicAuth(cfg.Neo4J.Username, cfg.Neo4J.Password, ""),
	)
	if err != nil {
		logger.Error("create neo4j driver", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := driver.Close(ctx); err != nil {
			logger.Error("close neo4j driver", "error", err)
		}
	}()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		logger.Error("verify neo4j connectivity", "error", err)
		os.Exit(1)
	}

	waterlooClient := waterloo.NewClient(cfg.Waterloo.BaseURL, cfg.Waterloo.APIKey, cfg.Waterloo.HTTPTimeout, logger)
	store := neo4jstore.New(driver, cfg.Neo4J.Database, logger)
	syncService := syncer.New(waterlooClient, store, cfg.TermCodes, logger)
	serviceRunner := runner.New(syncService.Sync, cfg.SyncInterval, cfg.SyncTimeout, logger)

	if err := serviceRunner.Run(ctx); err != nil {
		logger.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}
