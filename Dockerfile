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

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/semops /usr/local/bin/semops

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/semops"]
