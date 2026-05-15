package runner

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoopRunsImmediatelyAndOnTick(t *testing.T) {
	trigger := NewManualTrigger(1)
	var calls atomic.Int32
	loop := NewLoop(func(context.Context) error {
		calls.Add(1)
		return nil
	}, trigger, time.Second, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		loop.Run(ctx)
	}()

	waitForCalls(t, &calls, 1)
	trigger.Tick()
	waitForCalls(t, &calls, 2)
	cancel()
	<-done
}

func TestLoopSkipsTickWhileSyncRunning(t *testing.T) {
	trigger := NewManualTrigger(1)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32

	loop := NewLoop(func(context.Context) error {
		calls.Add(1)
		close(started)
		<-release
		return nil
	}, trigger, time.Second, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		loop.Run(ctx)
	}()

	<-started
	trigger.Tick()
	time.Sleep(25 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1 while first sync is still running", got)
	}

	close(release)
	waitForCalls(t, &calls, 1)
	cancel()
	<-done
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func waitForCalls(t *testing.T, calls *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("calls = %d, want %d", calls.Load(), want)
		case <-ticker.C:
			if calls.Load() >= want {
				return
			}
		}
	}
}
