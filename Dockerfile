# Build stage
FROM golang:1.23.2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make build-base

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG VERSION=0.0.0
ARG USER=docker
ARG TIME

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X 'github.com/maxthom/mir/internal/libs/build_meta.Version=${VERSION}' \
    -X 'github.com/maxthom/mir/internal/libs/build_meta.User=${USER}' \
    -X 'github.com/maxthom/mir/internal/libs/build_meta.Time=${TIME}' \
    -s -w" \
    -o mir \
    cmds/mir/main.go

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 -S mir && \
    adduser -u 1000 -S mir -G mir

# Copy binary from builder
COPY --from=builder /build/mir /usr/local/bin/mir

# Create config directory
RUN mkdir -p /home/mir/.config/mir && \
    chown -R mir:mir /home/mir

# Switch to non-root user
USER mir

# Set working directory
WORKDIR /home/mir

# Expose default port
EXPOSE 3015

# Run mir
ENTRYPOINT ["mir"]
