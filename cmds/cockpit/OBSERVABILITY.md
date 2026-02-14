# Observability Guide

This document provides a comprehensive guide to monitoring and observing the Cockpit server.

## Overview

The Cockpit server includes built-in observability features:

1. **Structured Logging** - Request/response logging with zerolog
2. **Prometheus Metrics** - HTTP metrics for monitoring
3. **Health Checks** - Readiness and liveness endpoints
4. **Go Profiling** - CPU and memory profiling via pprof

## Structured Logging

### Log Format

All HTTP requests are logged in JSON format with structured fields:

```json
{
  "level": "info",
  "method": "GET",
  "path": "/dashboard",
  "query": "tab=devices",
  "status": 200,
  "bytes": 1234,
  "duration_ms": 5,
  "remote_addr": "127.0.0.1:54321",
  "user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
  "referer": "http://localhost:3020/",
  "time": "2026-02-14T10:30:00Z",
  "message": "http request"
}
```

### Log Levels

The logging middleware automatically sets log levels based on HTTP status:

- **`info`**: 2xx and 3xx responses (successful requests)
- **`warn`**: 4xx responses (client errors - not found, bad request, etc.)
- **`error`**: 5xx responses (server errors)

### Configuration

Set log level via config file:

```yaml
logLevel: "info"  # Options: debug, info, warn, error
```

Or via command-line flag:

```bash
./bin/cockpit --debug  # Sets log level to debug
```

Or via environment variable:

```bash
MIR_LOGLEVEL=debug ./bin/cockpit
```

### Log Filtering and Analysis

#### Using jq

Filter logs by status code:

```bash
# Show only errors (5xx)
journalctl -u cockpit -f | jq 'select(.status >= 500)'

# Show slow requests (> 1 second)
journalctl -u cockpit -f | jq 'select(.duration_ms > 1000)'

# Show requests to specific path
journalctl -u cockpit -f | jq 'select(.path == "/api/devices")'
```

#### Using grep

```bash
# Show only error logs
journalctl -u cockpit -f | grep '"level":"error"'

# Show POST requests
journalctl -u cockpit -f | grep '"method":"POST"'
```

## Prometheus Metrics

### Available Metrics

#### Request Counter

```
cockpit_http_requests_total{method="GET", path="/", status="200"}
```

Tracks total number of requests by method, path, and status code.

**Use cases:**
- Calculate request rate
- Track error rates
- Monitor traffic patterns

#### Request Duration

```
cockpit_http_request_duration_seconds{method="GET", path="/", status="200"}
```

Histogram of request durations with buckets: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s.

**Use cases:**
- Calculate percentiles (p50, p95, p99)
- Identify slow endpoints
- Set SLA alerts

#### Request Size

```
cockpit_http_request_size_bytes{method="POST", path="/api/data"}
```

Histogram of incoming request sizes.

**Use cases:**
- Monitor upload sizes
- Detect anomalous large requests
- Capacity planning

#### Response Size

```
cockpit_http_response_size_bytes{method="GET", path="/", status="200"}
```

Histogram of outgoing response sizes.

**Use cases:**
- Monitor bandwidth usage
- Optimize large responses
- Detect cache effectiveness

#### In-Flight Requests

```
cockpit_http_requests_in_flight
```

Current number of requests being processed.

**Use cases:**
- Monitor concurrent load
- Detect request queuing
- Set concurrency limits

### Prometheus Configuration

Add Cockpit to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'cockpit'
    static_configs:
      - targets: ['localhost:3020']
    scrape_interval: 15s
```

### Example PromQL Queries

#### Request Rate

```promql
# Overall request rate (requests per second)
rate(cockpit_http_requests_total[5m])

# Request rate by path
sum by (path) (rate(cockpit_http_requests_total[5m]))

# Request rate by status code
sum by (status) (rate(cockpit_http_requests_total[5m]))
```

#### Response Time

```promql
# 50th percentile (median) response time
histogram_quantile(0.50, rate(cockpit_http_request_duration_seconds_bucket[5m]))

# 95th percentile response time
histogram_quantile(0.95, rate(cockpit_http_request_duration_seconds_bucket[5m]))

# 99th percentile response time
histogram_quantile(0.99, rate(cockpit_http_request_duration_seconds_bucket[5m]))

# Average response time
rate(cockpit_http_request_duration_seconds_sum[5m]) / rate(cockpit_http_request_duration_seconds_count[5m])
```

#### Error Rate

```promql
# 5xx error rate
sum(rate(cockpit_http_requests_total{status=~"5.."}[5m])) / sum(rate(cockpit_http_requests_total[5m]))

# 4xx error rate
sum(rate(cockpit_http_requests_total{status=~"4.."}[5m])) / sum(rate(cockpit_http_requests_total[5m]))

# Total error rate (4xx + 5xx)
sum(rate(cockpit_http_requests_total{status=~"[45].."}[5m])) / sum(rate(cockpit_http_requests_total[5m]))
```

#### Throughput

```promql
# Incoming traffic (bytes per second)
rate(cockpit_http_request_size_bytes_sum[5m])

# Outgoing traffic (bytes per second)
rate(cockpit_http_response_size_bytes_sum[5m])

# Total bandwidth
rate(cockpit_http_request_size_bytes_sum[5m]) + rate(cockpit_http_response_size_bytes_sum[5m])
```

#### Concurrency

```promql
# Current in-flight requests
cockpit_http_requests_in_flight

# Max in-flight requests over 5 minutes
max_over_time(cockpit_http_requests_in_flight[5m])

# Average in-flight requests
avg_over_time(cockpit_http_requests_in_flight[5m])
```

## Grafana Dashboard

### Dashboard JSON

Create a Grafana dashboard with the following panels:

#### Panel 1: Request Rate

```json
{
  "title": "Request Rate",
  "targets": [{
    "expr": "sum(rate(cockpit_http_requests_total[5m])) by (status)"
  }],
  "type": "graph"
}
```

#### Panel 2: Response Time Percentiles

```json
{
  "title": "Response Time",
  "targets": [
    {
      "expr": "histogram_quantile(0.50, rate(cockpit_http_request_duration_seconds_bucket[5m]))",
      "legendFormat": "p50"
    },
    {
      "expr": "histogram_quantile(0.95, rate(cockpit_http_request_duration_seconds_bucket[5m]))",
      "legendFormat": "p95"
    },
    {
      "expr": "histogram_quantile(0.99, rate(cockpit_http_request_duration_seconds_bucket[5m]))",
      "legendFormat": "p99"
    }
  ],
  "type": "graph"
}
```

#### Panel 3: Error Rate

```json
{
  "title": "Error Rate",
  "targets": [{
    "expr": "sum(rate(cockpit_http_requests_total{status=~\"5..\"}[5m])) / sum(rate(cockpit_http_requests_total[5m]))"
  }],
  "type": "graph"
}
```

#### Panel 4: In-Flight Requests

```json
{
  "title": "In-Flight Requests",
  "targets": [{
    "expr": "cockpit_http_requests_in_flight"
  }],
  "type": "graph"
}
```

## Alerting

### Prometheus Alerting Rules

Create alerts in `alerts.yml`:

```yaml
groups:
  - name: cockpit
    interval: 30s
    rules:
      # High error rate
      - alert: CockpitHighErrorRate
        expr: |
          sum(rate(cockpit_http_requests_total{status=~"5.."}[5m]))
          / sum(rate(cockpit_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cockpit error rate is above 5%"
          description: "{{ $value | humanizePercentage }} of requests are returning 5xx errors"

      # Slow response time
      - alert: CockpitSlowResponses
        expr: |
          histogram_quantile(0.95, rate(cockpit_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cockpit 95th percentile response time is above 1s"
          description: "95th percentile response time is {{ $value }}s"

      # Service down
      - alert: CockpitDown
        expr: up{job="cockpit"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Cockpit is down"
          description: "Cockpit has been down for more than 1 minute"

      # High concurrent load
      - alert: CockpitHighConcurrency
        expr: cockpit_http_requests_in_flight > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Cockpit has high number of concurrent requests"
          description: "{{ $value }} requests are currently in-flight"
```

## Health Checks

### Endpoint

```
GET /health
```

**Response:**
```json
{
  "status": "healthy"
}
```

**Status codes:**
- `200 OK`: Service is healthy and ready
- `503 Service Unavailable`: Service is unhealthy

### Usage

#### Kubernetes Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 3020
  initialDelaySeconds: 5
  periodSeconds: 10
```

#### Docker Healthcheck

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3020/health || exit 1
```

## Go Profiling

### Enable Profiling

Set the `GOPPROF` environment variable:

```bash
GOPPROF=1 ./bin/cockpit
```

### Available Profiles

- **CPU Profile**: `GET /debug/pprof/profile?seconds=30`
- **Memory Profile**: `GET /debug/pprof/heap`
- **Goroutine Profile**: `GET /debug/pprof/goroutine`
- **Block Profile**: `GET /debug/pprof/block`
- **Mutex Profile**: `GET /debug/pprof/mutex`

### Usage

```bash
# CPU profile (30 seconds)
curl http://localhost:3020/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profile
curl http://localhost:3020/debug/pprof/heap > mem.prof
go tool pprof mem.prof

# Goroutine profile
curl http://localhost:3020/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

## Best Practices

1. **Log Sampling**: In high-traffic environments, consider sampling logs to reduce volume
2. **Metric Cardinality**: Avoid high-cardinality labels (like user IDs) in metrics
3. **Alert Thresholds**: Set alert thresholds based on your SLAs and historical data
4. **Dashboard Organization**: Group related metrics together for easier analysis
5. **Retention**: Configure appropriate retention periods for logs and metrics
6. **Monitoring the Monitor**: Set up alerts for Prometheus and Grafana health

## Troubleshooting

### No metrics appearing

1. Check Prometheus is scraping:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Verify metrics endpoint works:
   ```bash
   curl http://localhost:3020/metrics
   ```

### Logs not appearing

1. Check log level configuration
2. Verify zerolog is configured correctly
3. Check systemd journal if using systemd

### High cardinality warnings

If you see high cardinality in metrics, review `sanitizePath()` function in `metrics.go` to ensure paths are properly grouped.

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Zerolog Documentation](https://github.com/rs/zerolog)
- [Go pprof](https://pkg.go.dev/net/http/pprof)
