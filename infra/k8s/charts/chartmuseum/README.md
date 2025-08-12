# Museum Helm Chart

A Helm chart for deploying ChartMuseum with DigitalOcean Spaces (S3-compatible) storage backend.

## Usage

### Upload a chart

```bash
# Package
cd mychart/
helm package .

# With basic auth enabled
curl -u admin:password --data-binary "@mychart-0.1.0.tgz" https://charts.example.com/api/charts

# With port-forward
kubectl port-forward svc/museum-chartmuseum 8080:8080
curl -u admin:password --data-binary "@mychart-0.1.0.tgz" http://localhost:8080/api/charts
```

### Add repository to Helm

```bash
# With basic auth
helm repo add mir-charts https://admin:password@museum-local:8081

# Without auth (if AUTH_ANONYMOUS_GET=true)
helm repo add mir-charts https://museum-local

helm repo update
```

### Search and install charts

```bash
helm search repo mir-charts/
helm install <name> mir-charts/mir
```

## Security Considerations

1. **Never commit credentials** - Always use Kubernetes secrets
2. **Enable HTTPS** - Use ingress with TLS certificates
3. **Set strong passwords** - Use complex passwords for basic auth
4. **Limit API access** - Consider setting `DISABLE_API=true` for read-only repositories
5. **Configure CORS** - Restrict `CORS_ALLOW_ORIGIN` to specific domains

## Backup and Recovery

Since charts are stored in DigitalOcean Spaces:
1. Enable versioning on your Spaces bucket
2. Set up lifecycle policies for old versions
3. Use DigitalOcean's built-in backup features
4. Consider replicating to another region for disaster recovery

## Monitoring

ChartMuseum exposes Prometheus metrics at `/metrics` when `DISABLE_METRICS=false`.

### Useful metrics:
- `chartmuseum_charts_served_total` - Total number of charts served
- `chartmuseum_charts_uploaded_total` - Total number of charts uploaded
- `chartmuseum_request_duration_seconds` - Request duration histogram

## Troubleshooting

### Check pod logs
```bash
kubectl logs -l app.kubernetes.io/name=museum
```

### Verify DigitalOcean Spaces connectivity
```bash
kubectl exec -it deployment/museum-chartmuseum -- sh
# Inside the pod
curl -I https://nyc3.digitaloceanspaces.com
```

### Common Issues

1. **403 Forbidden on upload** - Check AWS credentials and bucket permissions
2. **Connection refused** - Verify STORAGE_AMAZON_ENDPOINT matches your region
3. **Chart not found** - Ensure chart was uploaded successfully and index was regenerated
4. **Slow performance** - Enable Redis caching with `CACHE=redis`

## License

This chart is provided as-is under the MIT License.
