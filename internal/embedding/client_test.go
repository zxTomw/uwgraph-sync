package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAICompatibleClientOrdersAndValidatesEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer test-key"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "test-model" {
			t.Fatalf("model = %v, want test-model", request["model"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"index": 1, "embedding": []float32{0, 1}},
				{"index": 0, "embedding": []float32{1, 0}},
			},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL, "test-key", "test-model", 2, time.Second)
	embeddings, err := client.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	if embeddings[0][0] != 1 || embeddings[1][1] != 1 {
		t.Fatalf("embeddings = %v, want response ordered by index", embeddings)
	}
}

func TestOpenAICompatibleClientRejectsWrongDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"index": 0, "embedding": []float32{1}}},
		})
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(server.URL, "test-key", "test-model", 2, time.Second)
	if _, err := client.Embed(context.Background(), []string{"first"}); err == nil {
		t.Fatal("Embed returned nil error for wrong dimensions")
	}
}
