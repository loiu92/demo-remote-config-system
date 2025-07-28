# Remote Configuration System Helm Chart

This Helm chart deploys the Remote Configuration System with all its components on Kubernetes.

## Components

- **API Service**: Go-based REST API for configuration management
- **Demo App**: React-based demo application (ShopFlow Lite)
- **Dashboard**: Admin dashboard and SSE demo pages
- **PostgreSQL**: Primary database for configuration storage
- **Redis**: Caching layer for improved performance
- **Nginx**: Reverse proxy for routing and load balancing

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

After deployment, the application will be available through the Nginx service:

- **LoadBalancer** (default): Get external IP with `kubectl get svc remote-config-system-nginx`
- **Ingress**: Access via configured hostname
- **Port Forward**: `kubectl port-forward svc/remote-config-system-nginx 8080:80`

### Application URLs

- **Demo App**: `http://your-domain/demo/`
- **Dashboard**: `http://your-domain/dashboard`
- **SSE Demo**: `http://your-domain/demo/sse`
- **API**: `http://your-domain/api/`

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
