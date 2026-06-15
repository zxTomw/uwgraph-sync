.DEFAULT_GOAL := help

IMAGE ?= uwgraph-sync:local
TEST_PROJECT ?= uwgraph-sync-test
TEST_ENV = WATERLOO_API_KEY=not-used NEO4J_USERNAME=neo4j NEO4J_PASSWORD=integration-password UWGRAPH_TERM_CODES=1265 NEO4J_HTTP_PORT=0 NEO4J_BOLT_PORT=0

.PHONY: help up down logs build test integration-test integration-clean fmt-check tidy-check vet check docker-config verify

help:
	@printf '%s\n' \
		'make up                Build and run the app with Neo4j' \
		'make down              Stop the Compose stack' \
		'make logs              Follow app and Neo4j logs' \
		'make build             Build the production image' \
		'make test              Run unit tests with the race detector' \
		'make integration-test  Run Neo4j integration tests in Compose' \
		'make check             Run the fast credential-free validation gate' \
		'make verify            Run the full image and integration validation gate'

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs --follow

build:
	docker build --target runtime --tag $(IMAGE) .

test:
	go test -race -shuffle=on ./...

integration-test:
	@status=0; \
	$(TEST_ENV) docker compose -p $(TEST_PROJECT) --profile test up --build --abort-on-container-exit --exit-code-from integration-test integration-test || status=$$?; \
	$(MAKE) --no-print-directory integration-clean; \
	exit $$status

integration-clean:
	$(TEST_ENV) docker compose -p $(TEST_PROJECT) --profile test down --volumes

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		printf 'Files require gofmt:\n%s\n' "$$files"; \
		exit 1; \
	fi

tidy-check:
	go mod tidy -diff

vet:
	go vet ./...

docker-config:
	$(TEST_ENV) docker compose config --quiet

check: fmt-check tidy-check vet test docker-config

verify: check build integration-test
