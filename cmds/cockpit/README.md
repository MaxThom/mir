# Cockpit - Mir IoT Hub Web UI

A Go-based web server that serves the Mir Cockpit dashboard - a Svelte web application for managing and monitoring IoT devices.

## Overview

The Cockpit server embeds and serves the static files built from the Svelte web application located at `internal/ui/web`. It provides a modern, responsive interface for interacting with the Mir IoT Hub.

## Features

- **Static File Serving**: Embeds the built Svelte application for single-binary deployment
- **SPA Routing**: Proper client-side routing support with fallback to index.html
- **HTTP/2 Support**: Includes h2c (HTTP/2 over cleartext) for improved performance
- **Health Checks**: `/health` endpoint for monitoring
- **Prometheus Metrics**: Comprehensive HTTP metrics (requests, duration, size, in-flight)
- **Structured Logging**: Zerolog-based request/response logging with duration tracking
- **Hot Reload**: Air configuration for development with automatic reloading
- **Security Headers**: Content Security Policy, X-Frame-Options, and other security headers
- **CORS Support**: Configurable cross-origin resource sharing for API endpoints
- **Path Traversal Protection**: Security measures against directory traversal attacks

## Building

### Build Everything (Web UI + Go Server)

```bash
# Build both the Svelte web UI and Go server binary
just build-cockpit
```

### Build Components Separately

```bash
# Build only the Svelte web UI
just build-cockpit-web

# Build only the Go server binary
just build-cockpit-server
```

## Running

### Production Mode

Run the compiled server (requires building first):

```bash
just build-cockpit
./bin/cockpit
```

### Development Mode

**Option 1: Run Go server with hot-reload (Air)**

```bash
# Build the web UI first
just build-cockpit-web

# Run Go server with Air for hot-reload
just run-cockpit
```

**Option 2: Run Svelte dev server (for web UI development)**

```bash
# Run Vite dev server with HMR
just run-cockpit-web
```

> **Note**: During active web UI development, use `run-cockpit-web` to get Vite's hot module replacement. Once the UI is stable, build it and run the Go server.

## Configuration

The server can be configured via:

1. **Command-line flags**
2. **Environment variables** (prefix: `MIR_`)
3. **Configuration files**:
   - `/etc/mir/cockpit.yaml`
   - `~/.config/mir/cockpit.yaml`
   - Custom path via `--config` flag

### Default Configuration

```yaml
logLevel: "info"
httpServer:
  port: 3020
  allowedOrigins:
    - "http://localhost:5173"  # Svelte dev server
    - "http://localhost:3020"  # Self
```

**Note:** Leave `allowedOrigins` empty to allow all origins (development only). In production, specify exact origins.

### Example Usage

```bash
# Run with custom port
./bin/cockpit --config custom-config.yaml

# Run with debug logging
./bin/cockpit --debug

# Run with environment variable
MIR_HTTPSERVER_PORT=8080 ./bin/cockpit
```

## Endpoints

- `GET /` - Serves the Cockpit web application
- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics
- `GET /debug/pprof/*` - Go profiling (when `GOPPROF` env var is set)

## Security

The server implements several security best practices:

### Security Headers
- **Content-Security-Policy**: Restricts resource loading to prevent XSS attacks
- **X-Frame-Options**: Prevents clickjacking by denying iframe embedding
- **X-Content-Type-Options**: Prevents MIME-type sniffing
- **X-XSS-Protection**: Enables browser XSS protection
- **Referrer-Policy**: Controls referrer information
- **Permissions-Policy**: Restricts browser features (geolocation, camera, etc.)

### CORS Configuration
Configure allowed origins in your config file:

```yaml
httpServer:
  allowedOrigins:
    - "https://example.com"
    - "http://localhost:5173"
```

Leave empty for development to allow all origins (not recommended for production).

### Path Security
- Directory traversal protection
- Proper path cleaning and validation
- Secure file serving from embedded filesystem

## Observability

The Cockpit server provides comprehensive observability through structured logging and Prometheus metrics.

### Structured Logging

All HTTP requests are logged with structured fields using zerolog:

```json
{
  "level": "info",
  "method": "GET",
  "path": "/dashboard",
  "query": "",
  "status": 200,
  "bytes": 1234,
  "duration_ms": 5,
  "remote_addr": "127.0.0.1:54321",
  "user_agent": "Mozilla/5.0...",
  "referer": "",
  "message": "http request"
}
```

**Log Levels:**
- `info`: Successful requests (2xx, 3xx)
- `warn`: Client errors (4xx)
- `error`: Server errors (5xx)

**Viewing Logs:**
```bash
# Run with debug logging
./bin/cockpit --debug

# Or set log level in config
logLevel: "debug"

# View logs in production
journalctl -u cockpit -f
```

### Prometheus Metrics

The `/metrics` endpoint exposes the following metrics:

#### HTTP Request Metrics

**`cockpit_http_requests_total`** (Counter)
- Total number of HTTP requests
- Labels: `method`, `path`, `status`

**`cockpit_http_request_duration_seconds`** (Histogram)
- HTTP request duration in seconds
- Labels: `method`, `path`, `status`
- Buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 seconds

**`cockpit_http_request_size_bytes`** (Histogram)
- HTTP request size in bytes
- Labels: `method`, `path`
- Buckets: 100, 1K, 10K, 100K, 1M, 10M, 100M bytes

**`cockpit_http_response_size_bytes`** (Histogram)
- HTTP response size in bytes
- Labels: `method`, `path`, `status`
- Buckets: 100, 1K, 10K, 100K, 1M, 10M, 100M bytes

**`cockpit_http_requests_in_flight`** (Gauge)
- Current number of HTTP requests being processed

#### Example Queries

```promql
# Request rate by path
rate(cockpit_http_requests_total[5m])

# 95th percentile response time
histogram_quantile(0.95, rate(cockpit_http_request_duration_seconds_bucket[5m]))

# Error rate (5xx responses)
sum(rate(cockpit_http_requests_total{status=~"5.."}[5m])) / sum(rate(cockpit_http_requests_total[5m]))

# Average response size
rate(cockpit_http_response_size_bytes_sum[5m]) / rate(cockpit_http_response_size_bytes_count[5m])

# In-flight requests
cockpit_http_requests_in_flight
```

### Grafana Dashboard

Import the Cockpit dashboard from the monitoring directory or create your own with these panels:

1. **Request Rate**: `rate(cockpit_http_requests_total[5m])`
2. **Response Time (p50, p95, p99)**: Quantiles of `cockpit_http_request_duration_seconds`
3. **Error Rate**: 4xx and 5xx responses
4. **In-Flight Requests**: `cockpit_http_requests_in_flight`
5. **Throughput**: Request and response bytes per second

## Development Workflow

1. **Web UI Development**:
   ```bash
   cd internal/ui/web
   npm run dev
   ```
   This runs the Vite dev server on port 5173 with hot module replacement.

2. **Full Stack Development**:
   ```bash
   # Terminal 1: Build web UI
   just build-cockpit-web

   # Terminal 2: Run Go server with Air
   just run-cockpit
   ```

3. **Production Build**:
   ```bash
   just build-cockpit
   ./bin/cockpit
   ```

## Architecture

### Middleware Stack

Requests flow through the following middleware layers (outermost to innermost):

```
Request
   │
   ▼
┌─────────────────────────────────────┐
│  HTTP/2 Handler (h2c)               │  ← HTTP/2 support
└─────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────┐
│  CORS Middleware                    │  ← Cross-origin headers
└─────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────┐
│  Security Headers Middleware        │  ← CSP, X-Frame-Options, etc.
└─────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────┐
│  Logging Middleware                 │  ← Structured logging (zerolog)
└─────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────┐
│  Metrics Middleware                 │  ← Prometheus metrics
└─────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────┐
│  Router (ServeMux)                  │
└─────────────────────────────────────┘
   │
   ├─→ /health          → Health Check
   ├─→ /metrics         → Prometheus Metrics
   ├─→ /debug/pprof/*   → Go Profiling
   └─→ /*               → SPA Handler
                            │
                            ▼
                     ┌──────────────────┐
                     │  Embedded FS     │
                     │  Static Files    │
                     │  or index.html   │
                     └──────────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────┐
│     Cockpit Go Server (Port 3020)   │
│                                     │
│  Observability:                     │
│  • Structured Logging (zerolog)     │
│  • Prometheus Metrics               │
│  • Health Checks                    │
│                                     │
│  Security:                          │
│  • CORS                             │
│  • Security Headers                 │
│  • Path Traversal Protection        │
│                                     │
│  Serving:                           │
│  • HTTP/2 (h2c)                     │
│  • Embedded Svelte Build            │
│  • SPA Routing                      │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│   Svelte Web Application            │
│   (internal/ui/web/build/)          │
│                                     │
│  • SvelteKit 2.x                    │
│  • TailwindCSS 4.x                  │
│  • Shadcn-svelte Components         │
└─────────────────────────────────────┘
```

## Deployment

The server can be deployed as a single binary since it embeds all static assets:

```bash
# Build optimized binary
just build-cockpit

# Copy to server
scp bin/cockpit user@server:/usr/local/bin/

# Run on server
cockpit
```

## Troubleshooting

### Build directory not found

If you see an error about missing build directory:

```bash
# Make sure to build the web UI first
just build-cockpit-web
```

### Port already in use

If port 3020 is already in use:

```bash
# Use a different port via environment variable
MIR_HTTPSERVER_PORT=8080 ./bin/cockpit
```

### SPA routing not working

Make sure you're accessing the server directly (not through a proxy) or configure your proxy to forward all requests to the Cockpit server.
