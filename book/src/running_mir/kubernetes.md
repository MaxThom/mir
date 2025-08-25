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

### Create Image Pull Secret to access GitHub Container Registry

[Access to Mir GitHub Container Registry (ghcr.io)](../resources/access_mir_container_reg.md)

### Install Mir Chart

Create a custom values file `custom.values.yaml` to pass secrets, update ingress hosts, persistences and resources:

```yaml
imagePullSecrets:
  - name: ghcr-mir-secret

ingress:
  enabled: true
  className: ""
  annotations: {}
  hosts:
    - host: mir.local
      paths:
        - path: /
          pathType: Prefix
resources: {}
  # limits:
  #   cpu: 100m
  #   memory: 128Mi
  # requests:
  #   cpu: 100m
  #   memory: 128Mi

nats:
  enabled: true
  ingress:
    enabled: true
    className: ""
    annotations: {}
    host: nats-local
    path: /
    pathType: Prefix
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
# Install latest version
helm install mir mir/mir \
  --namespace mir \
  --create-namespace
  -f custom.values.yaml
```

Default values, includes:

- Load balancer on :31422
- Mir services
- NATS with JetStream
- SurrealDB
- InfluxDB
- Service Monitors
- Grafana Dashboards

### Access Mir

```bash
# Access the server using the CLI on localhost
mir tools config edit
# contexts:
#  - name: k8s
#    target: nats://<cluster_ip>:31422
#    grafana: <grafana_url>
mir ctx k8s

# Use
mir dev ls
```

## Deployment Scenarios

The chart includes several pre-configured values files for common deployment [scenarios](https://github.com/MaxThom/mir/tree/main/infra/k8s/charts/mir)

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

### 2. Standard Deployment (`values-standard.yaml`) (Recommended and Default)

Deploy Mir with all infrastructure components but without monitoring.

**Use when**: You need a production-ready IoT platform and already have a prometheus monitoring stack or else.

```bash
helm install mir ./mir -f values-standard.yaml
```

**Includes**:
- Mir services
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

## Security Considerations

### Using Secrets

For production deployments, use Kubernetes secrets for sensitive data:

1. Create secret files in `secret/` [directory](https://github.com/MaxThom/mir/tree/main/infra/k8s/charts/mir/secret):
   - `mir.secret.yaml` - Mir service credentials
   - `surreal.secret.yaml` - SurrealDB credentials
   - `influx.secret.yaml` - InfluxDB credentials

2. Apply secrets before installing the chart:
```bash
kubectl apply -f secret/
```

3. Update your `custom.values.yaml`
```yaml
## Mir
secretRef: mir-secret
## Surreal
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
## Influx
influxdb2:
  adminUser:
    existingSecret: influx-secret
```

## Next Steps

- Set up device connections to your Mir cluster
- Configure monitoring dashboards
- Explore the [Mir CLI](../operating_mir/mir_cli_tui.md) for cluster management
