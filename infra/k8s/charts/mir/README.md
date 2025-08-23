# Mir IoT Hub Helm Chart

A comprehensive Helm chart for deploying the Mir IoT Hub platform on Kubernetes. This chart provides flexible deployment options from minimal setups to full production-ready deployments with complete observability.

## Overview

Mir IoT Hub is a comprehensive IoT platform that enables secure communication between IoT devices and cloud services. This Helm chart packages:

- **Core Services**: Mir microservices for device management, telemetry, commands, and configuration
- **Infrastructure**: NATS messaging, SurrealDB for metadata, InfluxDB for time-series data
- **Observability**: Prometheus monitoring, Grafana dashboards, Loki log aggregation
- **High Availability**: Configurable replicas and clustering for production workloads

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure (for persistent storage)
- Optional: Ingress controller (nginx, traefik) for external access
- Optional: cert-manager for TLS certificate management

## Installation

### Add Helm Repository

```bash
# If using a Helm repository
helm repo add mir https://museum.mirhub.io/
helm repo update
```

## Deployment Scenarios

The chart includes several pre-configured values files for common deployment scenarios:

### 0. Default (`values.yaml`)

Deploy Mir with all infrastructure components but without monitoring.

Create loadbalancer for Nats on NodePort 31422.

**Includes**:
- Mir services
- NATS with JetStream
- SurrealDB
- InfluxDB
- Service Monitors
- Grafana Dashboards

### 1. Minimal Deployment (`values-minimal.yaml`)

Deploy only Mir services, connecting to external infrastructure.

**Use when**: You have existing NATS, SurrealDB, and InfluxDB instances.

```bash
helm install mir ./mir -f values-minimal.yaml
```

**Configuration required**:
- Update external service URLs in the config section
- Configure authentication credentials
- Adjust resource limits as needed

### 2. Standard Deployment (`values-standard.yaml`) (Recommended)

Deploy Mir with all infrastructure components but without monitoring.

**Use when**: You need a production-ready IoT platform and already have a prometheus monitoring stack or else.

```bash
helm install mir ./mir -f values-standard.yaml
```

**Includes**:
- Mir services (2 replicas for HA)
- NATS with JetStream (3-node cluster)
- SurrealDB with 20Gi storage
- InfluxDB with 50Gi storage

### 3. Full Deployment (`values-full.yaml`)

Complete deployment with all services and full observability stack.

**Use when**: You need a production-ready platform with complete monitoring and logging.

```bash
helm install mir ./mir -f values-full.yaml
```

**Includes**:
- Everything from Standard deployment
- Prometheus for metrics collection
- Grafana with pre-configured dashboards
- AlertManager for alerting
- Loki for log aggregation
- Promtail for log collection

### 5. Local Development (`values-k3d.yaml`)

Optimized for local Kubernetes development with k3d/kind.

```bash
helm install mir ./mir -f values-k3d.yaml
```

**Features**:
- Reduced resource requirements
- Traefik ingress configuration
- Shorter retention periods
- Local hostnames (*.local)

## Configuration

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Mir image repository | `ghcr.io/maxthom/mir` |
| `image.tag` | Mir image tag | `""` (uses chart appVersion) |
| `replicaCount` | Number of Mir replicas | `1` |
| `config.mir.logLevel` | Log level for Mir services | `info` |
| `config.mir.url` | NATS URL (auto-configured if nats.enabled) | `""` |
| `config.surreal.url` | SurrealDB URL (auto-configured if surrealdb.enabled) | `""` |
| `config.influx.url` | InfluxDB URL (auto-configured if influxdb2.enabled) | `""` |

### Infrastructure Components

| Component | Parameter | Description |
|-----------|-----------|-------------|
| NATS | `nats.enabled` | Enable NATS messaging |
| | `nats.config.cluster.replicas` | NATS cluster size |
| | `nats.config.jetstream.enabled` | Enable JetStream |
| SurrealDB | `surrealdb.enabled` | Enable SurrealDB |
| | `surrealdb.persistence.size` | Storage size |
| InfluxDB | `influxdb2.enabled` | Enable InfluxDB |
| | `influxdb2.persistence.size` | Storage size |
| | `influxdb2.adminUser.retention_policy` | Data retention period |

### Monitoring Stack

| Component | Parameter | Description |
|-----------|-----------|-------------|
| Prometheus | `kube-prometheus-stack.enabled` | Enable Prometheus stack |
| | `kube-prometheus-stack.prometheus.prometheusSpec.retention` | Metrics retention |
| Grafana | `kube-prometheus-stack.grafana.enabled` | Enable Grafana |
| | `kube-prometheus-stack.grafana.adminPassword` | Admin password |
| Loki | `loki.enabled` | Enable log aggregation |
| | `loki.loki.limits_config.retention_period` | Log retention |
| Promtail | `promtail.enabled` | Enable log collection |

## Security Considerations

### Using Secrets

For production deployments, use Kubernetes secrets for sensitive data:

1. Create secret files in `secret/` directory:
   - `mir.secret.yaml` - Mir service credentials
   - `surreal.secret.yaml` - SurrealDB credentials
   - `influx.secret.yaml` - InfluxDB credentials

2. Apply secrets before installing the chart:
```bash
kubectl apply -f secret/
```

3. Reference secrets in your values:
```yaml
secretRef: mir-secret
surrealdb:
  podExtraEnv:
    - name: SURREAL_PASS
      valueFrom:
        secretKeyRef:
          name: surreal-secret
          key: SURREAL_PASS
```

## Accessing Services

### Port Forwarding (Development)

```bash
# Mir API
kubectl port-forward svc/mir 3015:80

# Grafana
kubectl port-forward svc/mir-grafana 3000:80

# NATS
kubectl port-forward svc/mir-nats 4222:4222
```

### Ingress (Production)

Configure ingress in your values file:

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: mir.example.com
      paths:
        - path: /
          pathType: Prefix
```

### LoadBalancer for NATS

NATS requires TCP access and cannot use HTTP ingress:

```yaml
nats:
  service:
    merge:
      spec:
        type: LoadBalancer
        ports:
          - port: 4222
            nodePort: 31422  # Optional: fixed NodePort
```

## Monitoring and Dashboards

### Pre-configured Dashboards

The chart includes Grafana dashboards organized in folders:

- **Mir**: Core service metrics and telemetry
  - Device overview
  - Command execution
  - Configuration management
  - Telemetry flow
  - Event store

- **Mir Infra**: Infrastructure components
  - NATS server and JetStream metrics
  - InfluxDB performance
  - Go runtime metrics

- **Mir PromStack**: Monitoring stack health
  - Prometheus metrics
  - Grafana performance
  - Loki ingestion
  - AlertManager status

### Accessing Dashboards

1. Port forward to Grafana:
```bash
kubectl port-forward svc/mir-grafana 3000:80
```

2. Login with credentials (default: admin/mir-operator)

3. Navigate to Dashboards → Browse

### Mir Alerts

Add custom PrometheusRules in `values.yaml`:

```yaml
mirPrometheusRules:
  enabled: true
  labels:
    prometheus: mir
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a pull request with your changes
4. Update documentation as needed

## Support

- Documentation: [Mir Docs](https://mir-docs.example.com)
- Issues: [GitHub Issues](https://github.com/maxthom/mir-ecosystem/issues)
- Community: [Discord/Slack](https://community.example.com)

## License

This Helm chart is provided under the same license as the Mir IoT Hub project. See the LICENSE file in the repository root for details.
