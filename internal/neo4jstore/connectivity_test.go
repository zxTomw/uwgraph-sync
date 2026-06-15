package neo4jstore

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

type connectivityVerifierFunc func(context.Context) error

func (f connectivityVerifierFunc) VerifyConnectivity(ctx context.Context) error {
	return f(ctx)
}

func TestWaitForConnectivityRetriesUntilSuccessful(t *testing.T) {
	attempts := 0
	verifier := connectivityVerifierFunc(func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("not ready")
		}
		return nil
	})

	err := waitForConnectivity(
		context.Background(),
		verifier,
		time.Second,
		0,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("waitForConnectivity returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestWaitForConnectivityStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForConnectivity(
		ctx,
		connectivityVerifierFunc(func(context.Context) error {
			return errors.New("not ready")
		}),
		time.Second,
		time.Hour,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForConnectivity error = %v, want context.Canceled", err)
	}
}

func TestWaitForConnectivityReturnsStartupTimeout(t *testing.T) {
	err := waitForConnectivity(
		context.Background(),
		connectivityVerifierFunc(func(context.Context) error {
			return errors.New("not ready")
		}),
		time.Millisecond,
		time.Hour,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err == nil {
		t.Fatal("waitForConnectivity returned nil error")
	}
	if got, want := err.Error(), "neo4j connectivity not ready within 1ms: not ready"; got != want {
		t.Fatalf("waitForConnectivity error = %q, want %q", got, want)
	}
}
