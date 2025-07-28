# Remote Configuration System Helm Chart

This Helm chart deploys the Remote Configuration System with all its components on Kubernetes.

## Components

- **API Service**: Go-based REST API for configuration management (LoadBalancer on port 8080)
- **Demo App**: React-based demo application with nginx (LoadBalancer on port 3000)
- **Dashboard**: Admin dashboard and SSE demo pages with nginx (LoadBalancer on port 4000)
- **PostgreSQL**: Primary database for configuration storage
- **Redis**: Caching layer for improved performance

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PersistentVolume provisioner support in the underlying infrastructure (for PostgreSQL and Redis persistence)

## Installation

### Using Helm directly

1. **Install the chart**:
   ```bash
   helm install remote-config-system ./helm/remote-config-system
   ```

2. **Install with custom values**:
   ```bash
   helm install remote-config-system ./helm/remote-config-system -f custom-values.yaml
   ```

3. **Upgrade the deployment**:
   ```bash
   helm upgrade remote-config-system ./helm/remote-config-system
   ```

### Using ArgoCD (GitOps)

1. **Apply the ArgoCD Application**:
   ```bash
   kubectl apply -f argocd/remote-config-system-app.yaml
   ```

2. **Access ArgoCD UI** and sync the application, or use CLI:
   ```bash
   argocd app sync remote-config-system
   ```

## Configuration

### Default Values

The chart comes with sensible defaults in `values.yaml`. Key configurations include:

- **Images**: Uses GitHub Container Registry images
- **Resources**: Conservative CPU/memory limits
- **Persistence**: 8Gi for PostgreSQL, 4Gi for Redis
- **Service Types**: ClusterIP for internal services, LoadBalancer for Nginx
- **Health Checks**: Enabled for all services

### Common Customizations

#### 1. Image Tags
```yaml
api:
  image:
    tag: "v1.2.0"
demoApp:
  image:
    tag: "v1.2.0"
dashboard:
  image:
    tag: "v1.2.0"
```

#### 2. Resource Limits
```yaml
api:
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 500m
      memory: 512Mi
```

#### 3. Persistence
```yaml
postgresql:
  persistence:
    enabled: true
    size: 20Gi
    storageClass: "fast-ssd"
```

#### 4. Ingress (instead of LoadBalancer)
```yaml
nginx:
  service:
    type: ClusterIP

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: remote-config.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: remote-config-tls
      hosts:
        - remote-config.yourdomain.com
```

#### 5. Database Configuration
```yaml
postgresql:
  auth:
    database: remote_config
    username: postgres
    password: "your-secure-password"
```

## Accessing the Application

After deployment, each service has its own LoadBalancer for direct access:

```bash
# Get external IPs for all services
kubectl get svc -l app.kubernetes.io/name=remote-config-system
```

### Service Access

- **Demo App**: `http://<demo-app-external-ip>:3000/`
- **Dashboard**: `http://<dashboard-external-ip>:4000/dashboard`
- **SSE Demo**: `http://<dashboard-external-ip>:4000/demo/sse`
- **API**: `http://<api-external-ip>:8080/api/`

## Monitoring and Troubleshooting

### Check Pod Status
```bash
kubectl get pods -l app.kubernetes.io/name=remote-config-system
```

### View Logs
```bash
kubectl logs -l app.kubernetes.io/component=api
kubectl logs -l app.kubernetes.io/component=nginx
```

### Health Checks
All services have health check endpoints:
- API: `/health`
- Demo App: `/health`
- Dashboard: `/health`
- Nginx: `/health`

## Uninstallation

### Helm
```bash
helm uninstall remote-config-system
```

### ArgoCD
```bash
kubectl delete -f argocd/remote-config-system-app.yaml
```

## Development

### Local Testing
```bash
# Lint the chart
helm lint ./helm/remote-config-system

# Dry run
helm install --dry-run --debug remote-config-system ./helm/remote-config-system

# Template rendering
helm template remote-config-system ./helm/remote-config-system
```

### Values Schema
The chart includes validation for common configuration errors. Check the `values.yaml` file for all available options and their descriptions.
