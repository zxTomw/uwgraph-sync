# uwgraph

Binary-only Go service that periodically syncs selected University of Waterloo OpenData API v3 academic data into Neo4j.

## Data Ingested

- Terms
- Subjects
- Academic organizations
- Building locations
- Courses and course offerings for configured terms
- Class sections and class meetings for configured terms
- Exam schedules for configured terms

Important dates, holidays, food services, and WCMS content are intentionally not fetched in v1.

## Configuration

Required:

- `WATERLOO_API_KEY`
- `NEO4J_USERNAME`
- `NEO4J_PASSWORD`
- `UWGRAPH_TERM_CODES`, comma-separated, for example `1251,1255`

Optional:

- `WATERLOO_BASE_URL`, default `https://openapi.data.uwaterloo.ca`
- `NEO4J_URI`, default `bolt://localhost:7687`
- `NEO4J_DATABASE`, default `neo4j`
- `UWGRAPH_SYNC_INTERVAL`, default `6h`
- `UWGRAPH_HTTP_TIMEOUT`, default `30s`
- `UWGRAPH_SYNC_TIMEOUT`, default `30m`

## Run

```sh
go run ./cmd/uwgraph
```

The service runs one sync immediately, then repeats on `UWGRAPH_SYNC_INTERVAL`. If a sync is still running when the next interval arrives, that tick is skipped and logged.

## Test

```sh
go test ./...
```

Neo4j integration tests are opt-in:

```sh
go test -tags=integration ./internal/neo4jstore
```
