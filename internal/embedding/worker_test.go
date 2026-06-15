package embedding

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"uwgraph/internal/knowledge"
)

type fakeEmbeddingStore struct {
	pending []knowledge.PendingDocument
	updates []knowledge.EmbeddingUpdate
}

func (s *fakeEmbeddingStore) EnsureVectorIndex(context.Context, int) error {
	return nil
}

func (s *fakeEmbeddingStore) PendingDocuments(context.Context, string, int) ([]knowledge.PendingDocument, error) {
	documents := s.pending
	s.pending = nil
	return documents, nil
}

func (s *fakeEmbeddingStore) ApplyEmbeddings(_ context.Context, updates []knowledge.EmbeddingUpdate) (int, error) {
	s.updates = append(s.updates, updates...)
	return len(updates), nil
}

type fakeProvider struct{}

func (fakeProvider) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	result := make([][]float32, len(inputs))
	for i := range inputs {
		result[i] = []float32{1, 0}
	}
	return result, nil
}

func TestWorkerPreservesContentHashInUpdates(t *testing.T) {
	store := &fakeEmbeddingStore{pending: []knowledge.PendingDocument{{
		DocumentKey: "course:CS 135",
		Text:        "Course text",
		ContentHash: "hash",
	}}}
	worker := NewWorker(
		store,
		fakeProvider{},
		"model",
		2,
		10,
		time.Minute,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }

	updated, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated = %d, want 1", updated)
	}
	if got := store.updates[0].ContentHash; got != "hash" {
		t.Fatalf("ContentHash = %q, want hash", got)
	}
	if got := store.updates[0].EmbeddedAt; !got.Equal(now) {
		t.Fatalf("EmbeddedAt = %s, want %s", got, now)
	}
}
