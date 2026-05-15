package runner

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

type Trigger interface {
	C() <-chan time.Time
	Stop()
}

type ManualTrigger struct {
	ch chan time.Time
}

func NewManualTrigger(buffer int) *ManualTrigger {
	return &ManualTrigger{ch: make(chan time.Time, buffer)}
}

func (t *ManualTrigger) C() <-chan time.Time {
	return t.ch
}

func (t *ManualTrigger) Stop() {
	close(t.ch)
}

func (t *ManualTrigger) Tick() {
	t.ch <- time.Now()
}

type Loop struct {
	syncFunc    SyncFunc
	trigger     Trigger
	syncTimeout time.Duration
	logger      *slog.Logger
	running     atomic.Bool
}

func NewLoop(syncFunc SyncFunc, trigger Trigger, syncTimeout time.Duration, logger *slog.Logger) *Loop {
	return &Loop{syncFunc: syncFunc, trigger: trigger, syncTimeout: syncTimeout, logger: logger}
}

func (l *Loop) Run(ctx context.Context) {
	defer l.trigger.Stop()
	l.startSync(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-l.trigger.C():
			if !ok {
				return
			}
			if !l.startSync(ctx) {
				l.logger.Warn("skipped sync tick because previous sync is still running")
			}
		}
	}
}

func (l *Loop) startSync(parent context.Context) bool {
	if !l.running.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer l.running.Store(false)
		ctx, cancel := context.WithTimeout(parent, l.syncTimeout)
		defer cancel()
		if err := l.syncFunc(ctx); err != nil {
			l.logger.Error("sync failed", "error", err)
		}
	}()
	return true
}
