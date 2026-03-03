ARG BUILDPLATFORM=linux/amd64
### Web build stage
# node 25 not available on arm7 and alpine
# https://hub.docker.com/_/node
FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder

WORKDIR /build

# WebSDK
COPY pkgs/web/package*.json ./pkgs/web/
COPY pkgs/web/vite.config.ts ./pkgs/web/
COPY pkgs/web/tsconfig.json ./pkgs/web/
RUN npm ci --prefix ./pkgs/web

COPY pkgs/web/src ./pkgs/web/src
RUN npm run build --prefix ./pkgs/web

# Cockpit
COPY internal/ui/web/package*.json ./internal/ui/web/
COPY internal/ui/web/svelte.config.js ./internal/ui/web/
COPY internal/ui/web/vite.config.ts ./internal/ui/web/
COPY internal/ui/web/tsconfig.json ./internal/ui/web/
RUN npm ci --prefix ./internal/ui/web

COPY internal/ui/web ./internal/ui/web
RUN npm run build --prefix ./internal/ui/web

#### Mir build stage
FROM --platform=$BUILDPLATFORM golang:1.26.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make build-base

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built web UI from web-builder stage
COPY --from=web-builder /build/internal/ui/web/build ./internal/ui/web/build

# Build arguments for multi-platform support
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=0.0.0
ARG USER=docker
ARG TIME

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-X 'github.com/maxthom/mir/internal/libs/build_meta.Version=${VERSION}' \
    -X 'github.com/maxthom/mir/internal/libs/build_meta.User=${USER}' \
    -X 'github.com/maxthom/mir/internal/libs/build_meta.Time=${TIME}' \
    -s -w" \
    -o mir \
    cmds/mir/main.go

### Mir runtime stage
FROM alpine:3.19

# Add metadata labels
LABEL org.opencontainers.image.title="Mir IoT Hub"
LABEL org.opencontainers.image.description="Comprehensive IoT platform for secure device communication, telemetry collection, and command execution"
LABEL org.opencontainers.image.authors="maxthom"
LABEL org.opencontainers.image.source="https://github.com/maxthom/mir"
LABEL org.opencontainers.image.documentation="https://book.mirhub.io"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.vendor="maxthom"

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
