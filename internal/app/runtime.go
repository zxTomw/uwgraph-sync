package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/config"
	"uwgraph/internal/neo4jstore"
)

func Logger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func LoadDotEnv(logger *slog.Logger) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Warn("load .env", "error", err)
	}
}

func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func OpenNeo4J(
	ctx context.Context,
	cfg config.Neo4JConfig,
	startupTimeout time.Duration,
	logger *slog.Logger,
) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, err
	}
	if err := neo4jstore.WaitForConnectivity(ctx, driver, startupTimeout, logger); err != nil {
		_ = driver.Close(context.Background())
		return nil, err
	}
	return driver, nil
}

func CloseNeo4J(driver neo4j.DriverWithContext, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := driver.Close(ctx); err != nil {
		logger.Error("close neo4j driver", "error", err)
	}
}
