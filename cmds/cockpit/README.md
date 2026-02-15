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
just build-cockpit-server
```

```bash
# Build only the Svelte web UI
just build-cockpit
```

## Running

### Local Development

For the best development experience with hot module replacement (HMR) and instant feedback:

**Step 1: Start the Go backend server**

```bash
# From project root
go run cmds/cockpit/main.go

# Or with hot-reload (Air)
just run-cockpit-server
```

The Go server runs on **port 3021** by default.

**Step 2: Start the Vite dev server (in a new terminal)**

```bash
# Navigate to web UI directory
cd internal/ui/web

# Start Vite dev server with HMR
npm run dev

# Or use justfile command
just run-cockpit
```

The Vite dev server runs on **port 5173** with automatic proxy to port 3021.

**Step 3: Open your browser**

Navigate to `http://localhost:5173`

**How it works:**
- Frontend runs on port 5173 with instant HMR (edit Svelte files and see changes immediately)
- API calls to `/api/*` are automatically proxied to the Go backend on port 3021
- No CORS issues, no manual configuration needed
- Your Svelte code uses relative URLs (`/api/...`) that work in both dev and production

**Simplified workflow with tmux/tmuxifier:**

```bash
# Start both servers in split panes
tmux new-session \; \
  send-keys 'go run cmds/cockpit/main.go' C-m \; \
  split-window -h \; \
  send-keys 'cd internal/ui/web && npm run dev' C-m
```

### Production Mode

Production uses a single server that serves both the static Svelte build and API endpoints:

**Step 1: Build everything**

```bash
# Build both web UI and Go server
just build-cockpit-server
```

This creates:
- `internal/ui/web/build/` - Static Svelte files (embedded in Go binary)
- `bin/cockpit` - Single executable with embedded frontend

**Step 2: Run the server**

```bash
./bin/cockpit
```

**Step 3: Open your browser**

Navigate to `http://localhost:3021`

**How it works:**
- Single server on port 3021 serves everything
- Static files served from embedded filesystem
- API routes handled by same server
- No proxy needed (same origin, no CORS)
- Single binary deployment

**Production deployment:**

```bash
# Copy binary to server
scp bin/cockpit user@server:/usr/local/bin/

# Run on server
cockpit

# Or with systemd
sudo systemctl start cockpit
```

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
  port: 3021
  allowedOrigins:
    - "http://localhost:5173"  # Svelte dev server (for development)
    - "http://localhost:3021"  # Self
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

### Recommended: Full Stack Development with HMR

Run both frontend and backend servers together:

```bash
# Terminal 1: Go backend on port 3021
go run cmds/cockpit/main.go

# Terminal 2: Vite dev server on port 5173 (with proxy)
cd internal/ui/web && npm run dev
```

Open `http://localhost:5173` for development with instant hot reload.

**What you get:**
- ⚡ Instant feedback - see Svelte changes immediately
- 🔄 Auto-refresh on file save
- 🌐 API proxy - calls to `/api/*` automatically route to Go backend
- 🚫 No CORS issues
- 💻 Same code works in production

### Web UI Only Development

If you're only working on the frontend and backend is stable:

```bash
cd internal/ui/web
npm run dev
```

Make sure the Go backend is running on port 3021 for API calls to work.

### Backend Only Development

If you're only changing Go code:

```bash
# Option 1: Manual restart
go run cmds/cockpit/main.go

# Option 2: Hot reload with Air
just run-cockpit
```

**Note:** You need to build the web UI first (`just build-cockpit-web`) for the embedded files to exist.

### Production Build & Test

Build and test the production bundle:

```bash
# Build everything
just build-cockpit

# Run production server
./bin/cockpit
```

Open `http://localhost:3021` to test the production build.

### Proxy Configuration

The Vite dev server is configured to proxy API requests:

**File:** `internal/ui/web/vite.config.ts`

```typescript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:3021',
      changeOrigin: true
    }
  }
}
```

This configuration:
- **Development**: Routes `/api/*` from port 5173 → 3021
- **Production**: Not used (single server, no proxy needed)

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
│     Cockpit Go Server (Port 3021)   │
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

If port 3021 is already in use:

```bash
# Use a different port via environment variable
MIR_HTTPSERVER_PORT=8080 ./bin/cockpit
```

**Note:** If you change the backend port, update the Vite proxy configuration in `internal/ui/web/vite.config.ts`:

```typescript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',  // Update to match your port
      changeOrigin: true
    }
  }
}
```

### SPA routing not working

Make sure you're accessing the server directly (not through a proxy) or configure your proxy to forward all requests to the Cockpit server.
