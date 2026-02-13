# Cockpit - Mir IoT Hub Web UI

A Go-based web server that serves the Mir Cockpit dashboard - a Svelte web application for managing and monitoring IoT devices.

## Overview

The Cockpit server embeds and serves the static files built from the Svelte web application located at `internal/ui/web`. It provides a modern, responsive interface for interacting with the Mir IoT Hub.

## Features

- **Static File Serving**: Embeds the built Svelte application for single-binary deployment
- **SPA Routing**: Proper client-side routing support with fallback to index.html
- **HTTP/2 Support**: Includes h2c (HTTP/2 over cleartext) for improved performance
- **Health Checks**: `/health` endpoint for monitoring
- **Metrics**: Prometheus-compatible `/metrics` endpoint
- **Hot Reload**: Air configuration for development with automatic reloading

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
```

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

```
┌─────────────────────────────────────┐
│     Cockpit Go Server (Port 3020)   │
│  ┌───────────────────────────────┐  │
│  │   HTTP/2 Handler (h2c)        │  │
│  └───────────────────────────────┘  │
│              ▼                       │
│  ┌───────────────────────────────┐  │
│  │   Health & Metrics Endpoints  │  │
│  └───────────────────────────────┘  │
│              ▼                       │
│  ┌───────────────────────────────┐  │
│  │   SPA Handler                 │  │
│  │   (Embedded Static Files)     │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│   Svelte Web Application            │
│   (internal/ui/web/build/)          │
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
