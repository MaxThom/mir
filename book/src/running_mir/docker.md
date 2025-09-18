# Docker & Docker Compose Deployment

Deploy Mir using Docker for containerized environments with flexible configuration options.

## Prerequisites

- Docker Engine 20.10+ or Docker Desktop
- Docker Compose v2.0+ (for multi-service deployments)
- [Access to Mir GitHub Repository](../resources/access_mir.md)
- [Access to Mir GitHub Container Registry (ghcr.io)](../resources/access_mir_container_reg.md)

## Docker Compose Deployment

The Compose comes with a full production setup:

- **Mir**: IoT Hub core service
- **NATS**: Message broker for inter-service communication
- **InfluxDB**: Time-series database for telemetry data
- **SurrealDB**: General database for device metadata
- **Prometheus Stack**: Monitoring and observability
  - Prometheus
  - Grafana
  - Loki
  - Promtail
  - Alertmanager

### Quick Start

The easiest way to get started is to download the pre-configured Docker Compose files from the latest Mir release:

 - [mir-compose.tar.gz](https://github.com/MaxThom/mir/releases/latest/)

```bash
# Extract
tar -vxf mir-compose.tar.gz

# Start the complete Mir stack
cd mir-compose/local-mir-support/
docker compose up -d

# Access the server using the CLI on localhost
mir tools config edit
# contexts:
#  - name: local
#    target: nats://localhost:4222
#    grafana: localhost:3000
mir ctx local

# Use
mir dev ls

## Stopping
docker compose down

## To stop and remove all data
docker compose down -v

# View logs
docker compose logs mir -f
```

## Configuration

The `.env` file in `local_mir_support/` contains the Mir version.
You can modify other settings in the individual compose files as needed.

- `ls -la` to see hidden files

### Environment Variables

Configure Mir using environment variables with the `MIR__` prefix:

| Variable | Description | Default |
|----------|-------------|---------|
| `MIR__NATS__URL` | NATS server URL | `nats://localhost:4222` |
| `MIR__NATS__TIMEOUT` | Connection timeout | `5s` |
| `MIR__SURREAL__URL` | SurrealDB WebSocket URL | `ws://localhost:8000` |
| `MIR__SURREAL__USER` | SurrealDB username | `root` |
| `MIR__SURREAL__PASSWORD` | SurrealDB password | `root` |
| `MIR__SURREAL__NAMESPACE` | SurrealDB namespace | `global` |
| `MIR__SURREAL__DATABASE` | SurrealDB database | `mir` |
| `MIR__INFLUX__URL` | InfluxDB HTTP URL | `http://localhost:8086` |
| `MIR__INFLUX__TOKEN` | InfluxDB auth token | - |
| `MIR__INFLUX__ORG` | InfluxDB organization | `Mir` |
| `MIR__INFLUX__BUCKET` | InfluxDB bucket | `mir` |
| `MIR__LOG_LEVEL` | Logging level | `info` |
| `MIR__PORT` | HTTP server port | `3015` |

### Configuration File

Mount a configuration file for advanced settings

Modify `mir-compose/mir/local-config.yaml`

```yaml
mir:
  url: "nats://local_mir_support-nats-1:4222"
  logLevel: "info"
  httpPort: 3015
surreal:
  url: "ws://local_mir_support-surrealdb-1:8000/rpc"
  namespace: "global"
  database: "mir"
  user: "root"
  password: "root"
influx:
  url: "http://local_mir_support-influxdb-1:8086/"
  token: "mir-operator-token"
  org: "Mir"
  bucket: "mir"
  batchSize: 1000
  flushInterval: 1000
  retryBufferLimit: 1073741824
  gzip: false
```

## Operating

### Port Exposures

```bash
# Grafana       <user>///<password>
localhost:3000 # admin///mir-operator
# InfluxDB
localhost:8086 # admin///mir-operator
# SurrealDB
localhost:8000 # root///root
# Prometheus
localhost:9090
# NatsIO
localhost:8222
```

### View Logs

```bash
# in mir-compose/local-mir-support/
# View Mir logs
docker compose logs mir

# Follow logs in real-time
docker compose logs -f mir
```

### Multi-Architecture Support

Mir Docker images support multiple architectures:

- `linux/amd64`: Intel/AMD 64-bit
- `linux/arm64`: ARM 64-bit
- `linux/arm32`: ARM 32-bit

Docker automatically selects the appropriate architecture.

### Security

Securing the environment is done via the NSC tool. Refer to [Security Tutorial](../security/auth-tutorial.md) for details.

## Next Steps

- Configure devices to connect to your Mir instance
- Set up monitoring dashboards in Grafana
- Review [Kubernetes deployment](./kubernetes.md) for production scale
- Explore the [Mir CLI](../operating_mir/mir_cli_tui.md) for management
