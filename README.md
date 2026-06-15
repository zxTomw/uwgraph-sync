# UW Graph - Sync

Go service that periodically syncs University of Waterloo academic data (courses, schedules, terms, etc.) into Neo4j.

## Data Ingested

- Terms
- Subjects
- Academic organizations
- Building locations
- Courses and course offerings for configured terms
- Class sections and class meetings for configured terms
- Exam schedules for configured terms

Important dates, holidays, food services, and WCMS content are intentionally not fetched in v1.

## Quick Start

Docker Compose is the primary development workflow. It builds the Go worker,
starts Neo4j, waits for the database to become healthy, and persists graph data
in a named volume.

```sh
cp .env.example .env
# Set WATERLOO_API_KEY and adjust the development password in .env.
make up
```

Neo4j Browser is available at `http://localhost:7474`. Stop the stack with
`Ctrl+C` or `make down`. Remove persisted database data with
`docker compose down --volumes`.

The service runs one sync immediately, then repeats on
`UWGRAPH_SYNC_INTERVAL`. A tick is skipped when the previous sync is still
running.

## Configuration

Required values:

- `WATERLOO_API_KEY`
- `NEO4J_USERNAME`
- `NEO4J_PASSWORD`
- `UWGRAPH_TERM_CODES`, comma-separated, for example `1251,1255`

Optional values and defaults are documented in `.env.example`. Notable runtime
settings include `UWGRAPH_SYNC_INTERVAL=6h`, `UWGRAPH_SYNC_TIMEOUT=30m`, and
`UWGRAPH_STARTUP_TIMEOUT=2m`. Existing environment variables take precedence
over values loaded from `.env`.

## Development Commands

```sh
make up                # Build and run the complete stack
make logs              # Follow app and Neo4j logs
make build             # Build the production image
make test              # Run unit tests with race detection
make integration-test  # Run tagged tests against Compose Neo4j
make check             # Run formatting, vet, tests, and Compose validation
```

For direct host debugging, start Neo4j separately and run:

```sh
go run ./cmd/uwgraph
```

The binary loads `.env` automatically and logs structured JSON to stdout.

## Container Design

The production image uses a multi-stage Go build and a `scratch` runtime. It
contains only the statically linked worker binary and CA certificates, runs as
non-root user `65532`, and exposes no port because the process is a scheduled
worker rather than an HTTP service.
