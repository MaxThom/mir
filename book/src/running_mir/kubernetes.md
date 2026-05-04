# Kubernetes Deployment

Deploy Mir on Kubernetes using Helm charts for production-ready, scalable IoT infrastructure.

Refer to the [repository `values.yaml`](https://github.com/MaxThom/mir/blob/main/infra/k8s/charts/mir/values.yaml) for all configuration options.

## Prerequisites

- Kubernetes cluster 1.24+
- kubectl configured to access your cluster
- Helm 3.8+
- (Optional) Ingress controller for external access
- (Optional) StorageClass for persistent volumes
- [Access to Mir GitHub Repository](../resources/access_mir.md)
- [Access to Mir GitHub Container Registry (ghcr.io)](../resources/access_mir_container_reg.md)

## Quick Start

```bash
# Add Mir Helm repository
helm repo add mir https://charts.mirhub.io
helm repo update
```

### Create Image Pull Secret

[Access to Mir GitHub Container Registry (ghcr.io)](../resources/access_mir_container_reg.md)

### Install Mir Chart

Create a `custom.values.yaml` with your environment-specific overrides:

```yaml
imagePullSecrets:
  - name: ghcr-mir-secret

# Configure contexts — tells the Cockpit UI how to reach NATS and Grafana
# Need at one context for this Mir Server
config:
  contexts:
    - name: "production"
      target: "nats://<cluster-ip>:31422"      # NATS TCP for CLI/devices
      webTarget: "ws://<nats-host>/nats-ws"    # NATS WebSocket for Cockpit
      grafana: "http://<grafana-host>"
      sec:
        credentials: ""   # NATS creds file content (leave empty for open)
        rootCA: ""
        tlsCert: ""
        tlsKey: ""
        password: ""      # Cockpit UI password (leave empty to disable)

ingress:
  enabled: true
  className: ""           # nginx, traefik, etc.
  annotations: {}
  hosts:
    - host: mir.example.com
      paths:
        - path: /
          pathType: Prefix

nats:
  enabled: true
  ingress:
    enabled: true
    className: ""
    annotations: {}
    host: nats.example.com   # NATS monitor endpoint
    path: /
    pathType: Prefix
    wsHost: mir.example.com  # Host for NATS WebSocket (can be same as Mir)
    wsPath: /nats-ws         # Path for NATS WebSocket via ingress
  service:
    merge:
      spec:
        type: LoadBalancer
        ports:
          - appProtocol: tcp
            name: nats
            nodePort: 31422
            port: 4222
            protocol: TCP
            targetPort: nats
          # NodePort for WebSocket, use this or ingress.wsHost/ingress.wsPath or both
          - appProtocol: tcp
            name: websocket
            nodePort: 31922
            port: 9222
            protocol: TCP
            targetPort: websocket
  config:
    jetstream:
      fileStore:
        pvc:
          size: 10Gi
  container: {}
  #   env:
  #     # Different from k8s units, suffix must be B, KiB, MiB, GiB, or TiB
  #     # Should be ~80% of memory limit
  #     GOMEMLIMIT: 6GiB
  #   merge:
  #     # Recommended minimum: at least 2 CPU cores and 8Gi memory for production JetStream clusters
  #     # Set same limit as request to ensure Guaranteed QoS
  #     resources: {}

surrealdb:
  enabled: true
  ingress:
    enabled: true
    className: ""
    annotations: {}
    hosts:
      - host: surreal.local
        paths:
          - path: /
            pathType: Prefix
    tls: []
  persistence:
    size: 10Gi
  resources: {}

influxdb2:
  enabled: true
  ingress:
    enabled: true
    className: ""
    annotations: {}
    hostname: influx.local
    path: /
    tls: false
  persistence:
    size: 10Gi
  resources: {}
```

### Install Chart

```bash
helm install mir mir/mir \
  --namespace mir \
  --create-namespace \
  -f custom.values.yaml
```

### Access Mir

```bash
# CLI access via NATS TCP
mir tools config edit
# Set target: nats://<cluster-ip>:31422
mir ctx production
mir dev ls

# Cockpit (web UI) — open in browser
# http://mir.example.com
```

## Deployment Scenarios

The chart includes several pre-configured values files for common deployment [scenarios](https://github.com/MaxThom/mir/tree/main/infra/k8s/charts/mir).

### 1. Minimal Deployment (`values-minimal.yaml`)

Deploy only Mir services, connecting to external infrastructure.

**Use when**: You have existing NATS, SurrealDB, and InfluxDB instances.

```bash
helm install mir ./mir -f values-minimal.yaml
```

**Configuration required**:
- Update external service URLs in the `config` section
- Configure authentication credentials
- Adjust resource limits as needed

### 2. Standard Deployment (`values-standard.yaml`) (Recommended)

Deploy Mir with all infrastructure components but without a monitoring stack.

**Use when**: You need a production-ready IoT platform and manage observability separately.

```bash
helm install mir ./mir -f values-standard.yaml
```

**Includes**:
- Mir services + Cockpit UI
- NATS with JetStream (3-node cluster)
- SurrealDB with 20Gi storage
- InfluxDB with 50Gi storage

### 3. Full Deployment (`values-full.yaml`)

Complete deployment with all services and full observability stack.

**Use when**: You need a self-contained platform with built-in monitoring and logging.

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

> **Note**: When `kube-prometheus-stack` is disabled, you must also disable the monitoring CRD resources to avoid `ServiceMonitor`/`PodMonitor` errors:
> ```yaml
> mirServiceMonitors:
>   enabled: false
> mirPrometheusRules:
>   enabled: false
> nats:
>   promExporter:
>     podMonitor:
>       enabled: false
> ```

## NATS WebSocket

The Cockpit UI connects to NATS via WebSocket. Two exposure options are available:

### Via NodePort (simpler)
Directly accessible on port `31922`. Set `webTarget: "ws://localhost:31922"` in your context config.

### Via Ingress (recommended for shared/production)
Route WebSocket through the ingress controller on the same host as the Cockpit UI. Uses path `/nats-ws` to differentiate from the Mir HTTP API:

```yaml
nats:
  ingress:
    enabled: true
    wsHost: mir.example.com  # Same host as Cockpit
    wsPath: /nats-ws
```

Set `webTarget: "ws://mir.example.com/nats-ws"` in your context config.

Traefik handles WebSocket upgrades natively — no additional annotations required.

## Security Considerations

### Using Secrets

For production deployments, use Kubernetes secrets for sensitive data:

1. Create secret files from the templates in `secret/` [directory](https://github.com/MaxThom/mir/tree/main/infra/k8s/charts/mir/secret):
   - `mir.secret.yaml` — Mir service credentials
   - `surreal.secret.yaml` — SurrealDB credentials
   - `influx.secret.yaml` — InfluxDB credentials

2. Apply secrets before installing:
```bash
kubectl apply -f secret/
```

3. Reference secrets in your `custom.values.yaml`:
```yaml
## Mir
secretRef: mir-secret

## SurrealDB
surrealdb:
  podExtraEnv:
    - name: SURREAL_USER
      valueFrom:
        secretKeyRef:
          name: surreal-secret
          key: SURREAL_USER
    - name: SURREAL_PASS
      valueFrom:
        secretKeyRef:
          name: surreal-secret
          key: SURREAL_PASS

## InfluxDB
influxdb2:
  adminUser:
    existingSecret: influx-secret
```

### Authentication and Authorization

Securing NATS with TLS and credentials is done via the NSC tool. Refer to [Security Tutorial](../security/auth.md) for setup details.

## Next Steps

- Set up device connections to your Mir cluster
- Configure monitoring dashboards
- Explore the [Mir CLI](../operating_mir/mir_cli_tui.md) for cluster management
