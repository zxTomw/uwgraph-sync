package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"uwgraph/internal/knowledge"
)

type Store interface {
	EnsureVectorIndex(context.Context, int) error
	PendingDocuments(context.Context, string, int) ([]knowledge.PendingDocument, error)
	ApplyEmbeddings(context.Context, []knowledge.EmbeddingUpdate) (int, error)
}

type Worker struct {
	store        Store
	provider     Provider
	model        string
	dimensions   int
	batchSize    int
	pollInterval time.Duration
	logger       *slog.Logger
	now          func() time.Time
}

func NewWorker(
	store Store,
	provider Provider,
	model string,
	dimensions int,
	batchSize int,
	pollInterval time.Duration,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		store:        store,
		provider:     provider,
		model:        model,
		dimensions:   dimensions,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		logger:       logger,
		now:          time.Now,
	}
}

func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	if err := w.store.EnsureVectorIndex(ctx, w.dimensions); err != nil {
		return 0, err
	}
	total := 0
	for {
		documents, err := w.store.PendingDocuments(ctx, w.model, w.batchSize)
		if err != nil {
			return total, err
		}
		if len(documents) == 0 {
			return total, nil
		}
		inputs := make([]string, len(documents))
		for i, document := range documents {
			inputs[i] = document.Text
		}
		vectors, err := w.provider.Embed(ctx, inputs)
		if err != nil {
			return total, fmt.Errorf("embed knowledge documents: %w", err)
		}
		if len(vectors) != len(documents) {
			return total, fmt.Errorf("embedding provider returned %d vectors for %d documents", len(vectors), len(documents))
		}
		embeddedAt := w.now().UTC()
		updates := make([]knowledge.EmbeddingUpdate, len(documents))
		for i, document := range documents {
			updates[i] = knowledge.EmbeddingUpdate{
				DocumentKey: document.DocumentKey,
				ContentHash: document.ContentHash,
				Model:       w.model,
				Embedding:   vectors[i],
				EmbeddedAt:  embeddedAt,
			}
		}
		updated, err := w.store.ApplyEmbeddings(ctx, updates)
		if err != nil {
			return total, err
		}
		total += updated
		w.logger.Info("embedded knowledge documents", "requested", len(documents), "updated", updated, "total", total)
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if _, err := w.RunOnce(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				w.logger.Error("embedding cycle failed", "error", err)
			}
		}
	}
}
