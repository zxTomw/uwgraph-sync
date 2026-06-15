# UW Graph

Docker-first Go services that sync University of Waterloo academic data into
Neo4j and expose it as a cited knowledge base for AI agents.

## Services

- `uwgraph`: periodically syncs terms, courses, organizations, locations,
  sections, instructors, meetings, and exams.
- `uwgraph-embed`: projects searchable documents and maintains Neo4j vector
  embeddings.
- `uwgraph-serve`: provides an authenticated REST API and Streamable HTTP MCP
  endpoint.

## Quick Start

Neo4j provides graph, full-text, and vector retrieval. No separate vector
database is required. You must provide Waterloo credentials and an
OpenAI-compatible embeddings endpoint.

```sh
cp .env.example .env
# Set WATERLOO_API_KEY, NEO4J_PASSWORD, embedding values, and API keys.
make up
```

Neo4j Browser is at `http://localhost:7474`; the knowledge service is at
`http://localhost:8080`. The first API start may retry while data is synced,
embedded, and indexes become online. Stop with `make down`.

## Configuration

Required values are documented in `.env.example`. Embedding dimensions must
match the selected model and become part of the Neo4j vector index definition.
After changing model dimensions, run `make rebuild-index`.

## Development Commands

```sh
make up                # Run Neo4j, sync, embedding, and knowledge API
make embed-once        # Backfill stale embeddings and exit
make rebuild-index     # Recreate the vector index and backfill it
make build             # Build the production image
make test              # Run unit tests with race detection
make integration-test  # Run tagged tests against Compose Neo4j
make check             # Run the fast credential-free validation gate
make verify            # Build and run every local/CI verification gate
```

For direct host debugging:

```sh
go run ./cmd/uwgraph
go run ./cmd/uwgraph-embed --once
go run ./cmd/uwgraph-serve
```

All binaries load `.env`, log structured JSON, and share one non-root
`scratch` image. See [Knowledge API](docs/knowledge-api.md) for REST/MCP usage.

## Repository References

- [AGENTS.md](AGENTS.md): canonical contribution and agent instructions.
- [Architecture](docs/architecture.md): package boundaries, lifecycle, and
  failure semantics.
- [Graph model](docs/graph-model.md): Neo4j identities, relationships, and
  schema-change checklist.
- [Knowledge API](docs/knowledge-api.md): retrieval, authentication, and agent
  integration.
