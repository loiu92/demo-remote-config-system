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
- `GET /api/events/{org}/{app}/{env}` - SSE stream for real-time updates

### Management API (admin)

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
