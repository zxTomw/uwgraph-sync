# Knowledge API

The knowledge server exposes cited Neo4j data to AI agents. It does not call an
LLM or generate final prose; an external agent runtime retrieves evidence over
REST or MCP and performs generation.

## Dependencies

- Neo4j 2026.05 provides graph, full-text, and vector retrieval.
- An OpenAI-compatible `/embeddings` endpoint provides document and query
  vectors.
- No separate vector database, queue, or cache is required for the current
  dataset and polling workload.

Set the embedding model and dimensions explicitly. Rebuild the vector index
after changing dimensions:

```sh
make rebuild-index
```

## Authentication

All `/v1/*` and `/mcp` requests require:

```text
Authorization: Bearer $UWGRAPH_KNOWLEDGE_API_KEY
```

`/healthz` and `/readyz` are unauthenticated. MCP requests with an `Origin`
header must match `UWGRAPH_MCP_ALLOWED_ORIGINS`; requests without `Origin` are
accepted for local non-browser clients.

## REST

```sh
curl -sS http://localhost:8080/v1/search \
  -H "Authorization: Bearer $UWGRAPH_KNOWLEDGE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"introductory programming","kinds":["course"],"limit":5}'
```

Endpoints:

- `POST /v1/search`
- `GET /v1/courses/{courseCode}`
- `GET /v1/courses/{courseCode}/offerings?termCode=...`
- `POST /v1/sections/search`
- `GET /v1/exams?termCode=...&sections=...`
- `GET /v1/buildings/{buildingCode}`

Search uses reciprocal-rank fusion over full-text and vector candidates.
Responses include stable entity URIs, source endpoints, sync timestamps, and
component scores.

## MCP

Connect a Streamable HTTP MCP client to `http://localhost:8080/mcp` with the
same bearer token. Available tools mirror the REST operations:
`search_catalog`, `get_course`, `list_course_offerings`, `search_sections`,
`list_exams`, and `get_building`. Resource templates expose
`uwgraph://courses/{courseCode}` and `uwgraph://terms/{termCode}`.
