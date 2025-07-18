# Use a minimal base with Go preinstalled
FROM golang:1.24-bullseye AS builder
RUN go version

# Set up work environment
WORKDIR /build

ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/go-build
ENV GOPROXY=https://proxy.golang.org,direct

# Use Docker BuildKit cache mount for Go modules
ADD ./go.work ./
ADD ./go.work.sum ./
ADD ./go.mod ./
ADD ./go.sum ./
ADD ./cmd/cribctl/go.mod ./cmd/cribctl/go.mod
ADD ./cmd/cribctl/go.sum ./cmd/cribctl/go.sum

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY ./ ./

# Build the Go binary
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    go build -o cribctl cmd/cribctl/*.go

# Build final image
FROM golang:1.24-bullseye AS final
COPY --from=builder /build/cribctl /usr/local/bin/cribctl

ENTRYPOINT ["cribctl"]
