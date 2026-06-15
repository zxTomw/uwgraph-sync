package neo4jstore

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const connectivityRetryInterval = 2 * time.Second

type ConnectivityVerifier interface {
	VerifyConnectivity(context.Context) error
}

func WaitForConnectivity(
	ctx context.Context,
	verifier ConnectivityVerifier,
	timeout time.Duration,
	logger *slog.Logger,
) error {
	return waitForConnectivity(ctx, verifier, timeout, connectivityRetryInterval, logger)
}

func waitForConnectivity(
	ctx context.Context,
	verifier ConnectivityVerifier,
	timeout time.Duration,
	retryInterval time.Duration,
	logger *slog.Logger,
) error {
	startupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := verifier.VerifyConnectivity(startupCtx); err == nil {
			if attempt > 1 {
				logger.Info("neo4j connectivity established", "attempts", attempt)
			}
			return nil
		} else {
			lastErr = err
			logger.Warn("waiting for neo4j connectivity", "attempt", attempt, "error", err)
		}

		timer := time.NewTimer(retryInterval)
		select {
		case <-startupCtx.Done():
			timer.Stop()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("neo4j connectivity not ready within %s: %w", timeout, lastErr)
		case <-timer.C:
		}
	}
}
