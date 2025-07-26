# Remote Configuration System

A real-time configuration management system built with Go, PostgreSQL, and Redis. This system allows applications to fetch and receive live updates of configuration settings without requiring redeployment.

## Features

- **Multi-tenant Architecture**: Organizations → Applications → Environments → Configurations
- **Real-time Updates**: Server-Sent Events for instant configuration changes
- **Version Control**: Track configuration history and rollback capabilities
- **Caching**: Redis-based caching for optimal performance
- **Web Dashboard**: Simple admin interface for configuration management
- **Demo Application**: Sample app demonstrating real-time config consumption

## Quick Start

### Prerequisites

- Docker and Docker Compose

### Running the System

1. Clone the repository:
```bash
git clone <repository-url>
cd remote-config-system
```

2. Start all services:
```bash
make up
```

3. Access the services:
- API: http://localhost:8080
- Demo App: http://localhost:3000
- Web Dashboard: http://localhost:8080/admin

### Development

Start only the database and Redis for local development:
```bash
make dev-up
```

View logs:
```bash
make logs
```

Stop all services:
```bash
make down
```

## API Endpoints

### Configuration API (for applications)
- `GET /config/{org}/{app}/{env}` - Get current configuration (public)
- `GET /api/config/{env}` - Get current configuration (API key required)

### Server-Sent Events (SSE) API
- `GET /events/{org}/{app}/{env}` - SSE stream for real-time configuration updates (public)
- `GET /api/events/{env}` - SSE stream for real-time configuration updates (API key required)

### Management API (admin)

#### Cache Management
- `GET /admin/cache/stats` - Get cache statistics and performance metrics
- `POST /admin/cache/warm` - Preload frequently accessed configurations into cache
- `DELETE /admin/cache` - Clear all cached configurations

#### SSE Management
- `GET /admin/sse/stats` - Get SSE statistics and connected clients information

#### Organization Management
- `GET /admin/orgs` - List all organizations
- `POST /admin/orgs` - Create a new organization
- `GET /admin/orgs/{org}` - Get organization details
- `PUT /admin/orgs/{org}` - Update organization
- `DELETE /admin/orgs/{org}` - Delete organization

#### Application Management
- `GET /admin/orgs/{org}/apps` - List applications in organization
- `POST /admin/orgs/{org}/apps` - Create a new application
- `GET /admin/orgs/{org}/apps/{app}` - Get application details
- `PUT /admin/orgs/{org}/apps/{app}` - Update application
- `DELETE /admin/orgs/{org}/apps/{app}` - Delete application

#### Environment Management
- `GET /admin/orgs/{org}/apps/{app}/envs` - List environments in application
- `POST /admin/orgs/{org}/apps/{app}/envs` - Create a new environment
- `GET /admin/orgs/{org}/apps/{app}/envs/{env}` - Get environment details
- `PUT /admin/orgs/{org}/apps/{app}/envs/{env}` - Update environment
- `DELETE /admin/orgs/{org}/apps/{app}/envs/{env}` - Delete environment

#### Configuration Management
- `PUT /admin/orgs/{org}/apps/{app}/envs/{env}/config` - Update configuration
- `GET /admin/orgs/{org}/apps/{app}/envs/{env}/history` - Get configuration version history
- `GET /admin/orgs/{org}/apps/{app}/envs/{env}/changes` - Get configuration change log
- `POST /admin/orgs/{org}/apps/{app}/envs/{env}/rollback` - Rollback to previous version

### API Usage Examples

#### Create an Organization
```bash
curl -X POST http://localhost:8080/admin/orgs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Company",
    "slug": "mycompany"
  }'
```

#### Create an Application
```bash
curl -X POST http://localhost:8080/admin/orgs/mycompany/apps \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Web App",
    "slug": "webapp"
  }'
```

#### Create an Environment
```bash
curl -X POST http://localhost:8080/admin/orgs/mycompany/apps/webapp/envs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production",
    "slug": "prod"
  }'
```

#### Update Configuration
```bash
curl -X PUT http://localhost:8080/admin/orgs/mycompany/apps/webapp/envs/prod/config \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "database_url": "postgres://localhost:5432/myapp",
      "api_key": "secret-key",
      "debug": false
    },
    "created_by": "admin@mycompany.com"
  }'
```

#### Get Configuration (Public)
```bash
curl http://localhost:8080/config/mycompany/webapp/prod
```

#### Get Configuration (API Key)
```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/config/prod
```

#### Cache Management
```bash
# Get cache statistics
curl http://localhost:8080/admin/cache/stats

# Warm cache with all configurations
curl -X POST http://localhost:8080/admin/cache/warm

# Clear all cache
curl -X DELETE http://localhost:8080/admin/cache
```

#### Server-Sent Events (Real-time Updates)
```bash
# Listen to configuration changes (public endpoint)
curl -N http://localhost:8080/events/mycompany/webapp/prod

# Listen to configuration changes (with API key)
curl -N -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/events/prod

# Get SSE statistics
curl http://localhost:8080/admin/sse/stats
```

#### JavaScript SSE Client Example
```javascript
// Connect to SSE stream
const eventSource = new EventSource('/events/mycompany/webapp/prod');

// Handle initial configuration
eventSource.addEventListener('initial_config', function(event) {
    const config = JSON.parse(event.data);
    console.log('Initial config:', config);
    updateApplicationConfig(config.config);
});

// Handle configuration updates
eventSource.addEventListener('config_update', function(event) {
    const update = JSON.parse(event.data);
    console.log('Config updated:', update);
    updateApplicationConfig(update.config);
});

// Handle rollbacks
eventSource.addEventListener('config_update', function(event) {
    const update = JSON.parse(event.data);
    if (update.action === 'rollback') {
        console.log('Config rolled back to version:', update.version);
        updateApplicationConfig(update.config);
    }
});

// Handle connection events
eventSource.addEventListener('connected', function(event) {
    console.log('Connected to SSE stream');
});

eventSource.addEventListener('ping', function(event) {
    console.log('Keep-alive ping received');
});

// Handle errors
eventSource.onerror = function(event) {
    console.error('SSE connection error:', event);
};
```

## Configuration

### Redis Caching Configuration

The system supports advanced Redis caching with the following environment variables:

```bash
# Redis connection
REDIS_HOST=localhost          # Redis host (default: localhost)
REDIS_PORT=6379              # Redis port (default: 6379)
REDIS_PASSWORD=              # Redis password (optional)
REDIS_DB=0                   # Redis database number (default: 0)

# Cache TTL settings
CACHE_TTL=300                # Default TTL in seconds (default: 300 = 5 minutes)
CACHE_SHORT_TTL=60           # Short TTL for frequently changing data (default: 60 = 1 minute)
CACHE_LONG_TTL=3600          # Long TTL for rarely changing data (default: 3600 = 1 hour)

# Cache features
CACHE_ENABLE_COMPRESSION=true # Enable compression for large configurations (default: false)
```

### Cache Features

- **Multi-tier TTL Strategy**: Different TTL values for different types of data
- **Automatic Compression**: Large configurations (>1KB) are automatically compressed
- **Cache Statistics**: Real-time metrics on cache hits, misses, and performance
- **Cache Warming**: Preload frequently accessed configurations on startup
- **Pattern-based Invalidation**: Efficient cache invalidation when configurations change
- **Fallback Support**: System continues to work even if Redis is unavailable

## Project Structure

```
remote-config-system/
├── cmd/api/                 # Main application entry point
├── internal/
│   ├── handlers/           # HTTP handlers
│   ├── services/           # Business logic
│   ├── models/             # Data models
│   ├── db/                 # Database operations
│   └── middleware/         # HTTP middleware
├── web/                    # Admin web interface
│   ├── static/             # CSS, JS files
│   └── templates/          # HTML templates
├── demo-app/               # Demo application
├── migrations/             # Database migrations
├── docker-compose.yml      # Docker services configuration
└── Dockerfile              # Application container
```

## Configuration Example

```json
{
  "maintenance": {
    "enabled": false,
    "message": "Scheduled maintenance in progress"
  },
  "features": {
    "dark_mode": true,
    "new_dashboard": false,
    "beta_features": true
  },
  "ui": {
    "theme_color": "#007bff",
    "max_items_per_page": 20,
    "show_footer": true
  },
  "limits": {
    "max_upload_size_mb": 10,
    "rate_limit_per_hour": 1000
  }
}
```

## Development Status

This is a portfolio project demonstrating:
- RESTful API design
- Real-time WebSocket/SSE implementation
- Database design and migrations
- Caching strategies
- Multi-tenant architecture
- Configuration versioning and rollback

## License

MIT License
