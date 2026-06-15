.DEFAULT_GOAL := help

.PHONY: help up down logs build test integration-test fmt-check vet check docker-config

help:
	@printf '%s\n' \
		'make up                Build and run the app with Neo4j' \
		'make down              Stop the Compose stack' \
		'make logs              Follow app and Neo4j logs' \
		'make build             Build the production image' \
		'make test              Run unit tests with the race detector' \
		'make integration-test  Run Neo4j integration tests in Compose' \
		'make check             Run formatting, vet, tests, and Compose validation'

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs --follow

build:
	docker compose build app

test:
	go test -race -shuffle=on ./...

integration-test:
	@status=0; \
	docker compose -p uwgraph-sync-test --profile test up --build --abort-on-container-exit --exit-code-from integration-test integration-test || status=$$?; \
	docker compose -p uwgraph-sync-test --profile test down --volumes; \
	exit $$status

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		printf 'Files require gofmt:\n%s\n' "$$files"; \
		exit 1; \
	fi

vet:
	go vet ./...

docker-config:
	docker compose config --quiet

check: fmt-check vet test docker-config
