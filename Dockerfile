# syntax=docker/dockerfile:1.6
#
# Build with Docker Compose so the local SemStreams checkout is available as a
# named build context:
#
#   docker compose -f compose.cop.yml build semops
#
# Direct docker build requires:
#
#   docker build --build-context semstreams=../semstreams -t semops:local .

ARG GO_VERSION=1.26.3

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

COPY --from=semstreams . /semstreams
COPY go.mod go.sum ./
RUN go mod edit -replace github.com/c360studio/semstreams=/semstreams
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_DATE=unknown

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT_SHA} \
      -X main.buildDate=${BUILD_DATE}" \
      -o /out/semops ./cmd/semops
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT_SHA} \
      -X main.buildDate=${BUILD_DATE}" \
      -o /out/semops-scenario-runner ./cmd/semops-scenario-runner
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT_SHA} \
      -X main.buildDate=${BUILD_DATE}" \
      -o /out/semops-feed-fixtures ./cmd/semops-feed-fixtures

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w \
      -X main.version=${VERSION} \
      -X main.commit=${COMMIT_SHA} \
      -X main.buildDate=${BUILD_DATE}" \
      -o /out/semops-klv-fixture ./cmd/semops-klv-fixture

FROM gcr.io/distroless/static-debian12:nonroot AS production

COPY --from=builder /out/semops /usr/local/bin/semops
COPY --from=builder /out/semops-scenario-runner /usr/local/bin/semops-scenario-runner
COPY --from=builder /out/semops-feed-fixtures /usr/local/bin/semops-feed-fixtures

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/semops"]

FROM debian:bookworm-slim AS media-tools

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/semops /usr/local/bin/semops
COPY --from=builder /out/semops-scenario-runner /usr/local/bin/semops-scenario-runner
COPY --from=builder /out/semops-feed-fixtures /usr/local/bin/semops-feed-fixtures
COPY --from=builder /out/semops-klv-fixture /usr/local/bin/semops-klv-fixture

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/semops"]
