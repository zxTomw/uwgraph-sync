//go:build integration

package neo4jstore

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/waterloo"
)

func TestStoreIntegrationEnsureSchemaAndUpsert(t *testing.T) {
	uri := os.Getenv("NEO4J_URI")
	user := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASSWORD")
	if uri == "" || user == "" || password == "" {
		t.Skip("NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD are required")
	}
	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}

	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		t.Fatalf("NewDriverWithContext: %v", err)
	}
	defer driver.Close(ctx)

	store := New(driver, database, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	if _, err := store.UpsertTerms(ctx, []waterloo.Term{{TermCode: "9999", Name: "Integration Test"}}); err != nil {
		t.Fatalf("UpsertTerms: %v", err)
	}
}
