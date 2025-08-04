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
helm install mir ./mir --values custom-values.yaml

# Check deployment
kubectl get pods
kubectl get svc
```

### 4. Access Mir

If using NodePort (default in values.yaml):
```bash
# Get the NodePort
kubectl get svc mir -o jsonpath='{.spec.ports[0].nodePort}'

# Access Mir (if k3d was created with port mapping)
curl http://localhost:3015/alive
```

## Uninstall

```bash
# Remove Mir
helm uninstall mir

# Delete cluster
k3d cluster delete mir-cluster
```
