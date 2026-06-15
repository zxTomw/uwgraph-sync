# Repository Guidelines

## Start Here

Read [README.md](README.md) for setup, then use
[docs/architecture.md](docs/architecture.md) and
[docs/graph-model.md](docs/graph-model.md) for behavioral contracts. Prefer
repository commands over ad hoc equivalents:

- `make check`: fast, credential-free validation for normal edits.
- `make verify`: full completion gate, including the image build and Neo4j
  integration test.
- `make up`: run the live sync stack; this requires a populated `.env`.

Do not run the live sync or call Waterloo APIs unless the task requires it.

## Code Map

- `cmd/uwgraph`: process wiring, configuration, signals, and lifecycle.
- `internal/config`: environment parsing and defaults.
- `internal/waterloo`: Waterloo API client and response models.
- `internal/syncer`: ingest orchestration and failure policy.
- `internal/graph`: stable graph identity-key construction.
- `internal/neo4jstore`: schema creation and parameterized Cypher writes.
- `internal/runner`: immediate and periodic sync scheduling.
- `uw-openapi/swagger.json`: read-only upstream API reference snapshot.

Keep `cmd/` thin. Define interfaces in the package that consumes them, pass
`context.Context` through I/O, and use structured `slog` attributes. Format Go
with `gofmt`; co-locate tests as `*_test.go`.

## Behavioral Invariants

- Waterloo read failures are fail-soft: log the failed dataset and continue
  independent work.
- Neo4j schema or write failures are fail-fast for the current sync.
- Only one sync may run at a time; overlapping ticks are skipped.
- Graph identity keys and relationship directions are compatibility contracts.
  Update graph code, Cypher, tests, and graph documentation together.
- Cypher values must remain parameters; never interpolate external data into
  query strings.

## Change Checklist

Add or update tests for observable behavior. Use the `integration` build tag
for tests requiring Neo4j. Before finishing:

1. Run `make check`.
2. Run `make verify` for changes affecting runtime, dependencies, containers,
   configuration, graph schema, or persistence.
3. Update the relevant documentation when behavior, commands, configuration,
   API mappings, or graph structure changes.

Never commit `.env`, credentials, local Neo4j data, or generated build output.
