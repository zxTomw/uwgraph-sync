# syntax=docker/dockerfile:1.7

FROM golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uwgraph ./cmd/uwgraph && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uwgraph-embed ./cmd/uwgraph-embed && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uwgraph-serve ./cmd/uwgraph-serve

FROM scratch AS runtime

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/uwgraph /uwgraph
COPY --from=build /out/uwgraph-embed /uwgraph-embed
COPY --from=build /out/uwgraph-serve /uwgraph-serve

USER 65532:65532

ENTRYPOINT ["/uwgraph"]
