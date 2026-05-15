package runner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type SyncFunc func(context.Context) error

type Runner struct {
	syncFunc    SyncFunc
	interval    time.Duration
	syncTimeout time.Duration
	logger      *slog.Logger
}

func New(syncFunc SyncFunc, interval, syncTimeout time.Duration, logger *slog.Logger) *Runner {
	return &Runner{
		syncFunc:    syncFunc,
		interval:    interval,
		syncTimeout: syncTimeout,
		logger:      logger,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	var running atomic.Bool
	var wg sync.WaitGroup

	r.startSync(ctx, &running, &wg)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("shutdown requested")
			wg.Wait()
			return nil
		case <-ticker.C:
			if !r.startSync(ctx, &running, &wg) {
				r.logger.Warn("skipped sync tick because previous sync is still running")
			}
		}
	}
}

func (r *Runner) startSync(parent context.Context, running *atomic.Bool, wg *sync.WaitGroup) bool {
	if !running.CompareAndSwap(false, true) {
		return false
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer running.Store(false)
		ctx, cancel := context.WithTimeout(parent, r.syncTimeout)
		defer cancel()
		if err := r.syncFunc(ctx); err != nil {
			r.logger.Error("sync failed", "error", err)
			return
		}
		r.logger.Info("sync completed")
	}()
	return true
}
