# Build stage
FROM golang:1.25.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /workspace

# Copy go mod files first (better caching)
COPY go.mod go.sum ./

# Use cache mount for go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build with cache mounts
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -a -trimpath -ldflags="-w -s" -o manager main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /workspace/manager /manager

USER 65532:65532

ENTRYPOINT ["/manager"]
