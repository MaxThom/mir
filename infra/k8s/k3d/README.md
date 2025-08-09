# Mir Kubernetes Deployment

This directory contains Kubernetes deployment configurations for Mir IoT Platform in local K3D.

## Quick Start with k3d

### 1. Create k3d Cluster

```bash
# Create cluster with registry config
k3d cluster create mir-local-dev --config k3d_config.yaml
```

### 3. Deploy Mir with Helm

```bash
# Install Mir
helm install mir ./mir

# Or with custom values
helm install mir . -f values-k3d.yaml

# Check deployment
kubectl get pods
kubectl get svc
```

### 4. Access MirStack

- NatsCluster on `localhost:31422`
- HTTP Services uses ingress on `localhost:8081`, to make it work locally, edit dns mapping file. On linux:

```sh
# sudo nvim /etc/hosts

127.0.0.1 localhost nats-local mir-local surreal-local
```

**Note on SurrealDB Persistence**: When using persistent storage, the PVC name in values must match the pattern `{release-name}-surrealdb-data`. For example, if installing with `helm install mir`, use `mir-surrealdb-data` as the claimName.

## Uninstall

```bash
# Remove Mir
helm uninstall mir

# Delete cluster
k3d cluster delete mir-local-dev
```
