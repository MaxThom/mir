# Build stage
ARG BUILDPLATFORM=linux/amd64
FROM --platform=$BUILDPLATFORM golang:1.26.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make build-base

ARG GITHUB_TOKEN
RUN if [ -n "${GITHUB_TOKEN}" ]; then \
    echo "Configuring private repo access..." && \
    go env -w GOPRIVATE=github.com/maxthom/mir && \
    git config --global url."https://${GITHUB_TOKEN}@github.com/maxthom/mir".insteadOf "https://github.com/maxthom/mir"; \
    fi

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for multi-platform support
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o mir.dev \
    cmd/main.go

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
COPY --from=builder /build/mir.dev /usr/local/bin/mir.dev

# Create config directory
RUN mkdir -p /etc/mir && \
    chown -R mir:mir /etc/mir

# Pass a base config for each deployment
# Override using extra config with volume mount
# docker run -v $(pwd)/device.instance.yaml:/home/mir/.config/mir/device.yaml mir.dev:latest
COPY device.yaml /etc/mir/device.yaml

# Switch to non-root user
USER mir

# Set working directory
WORKDIR /home/mir

# Run mir
ENTRYPOINT ["mir.dev"]
