# Router Sync

A Go-based router synchronization service that manages internet providers and routing policies using NATS.io as the source of truth. This service enables policy-based routing across multiple routers in a LAN environment.

## Features

- **Internet Provider Management**: Add, remove, and manage internet service providers with their associated network interfaces and routing tables
- **Policy-Based Routing**: Create routing policies based on source IP addresses (single IP or CIDR notation)
- **NATS.io Integration**: Uses NATS.io key-value store for persistent configuration storage
- **Real-time Synchronization**: Automatic synchronization between NATS KV store and router configuration
- **REST API**: Full CRUD operations for providers and policies
- **OpenAPI Documentation**: Auto-generated API documentation with Swagger
- **Prometheus Metrics**: Comprehensive metrics for monitoring
- **Netlink Integration**: Uses Linux netlink library for routing table management
- **Authentication Support**: NATS authentication with username/password or tokens
- **Persistent Storage**: Configuration survives reboots and power outages
- **Docker Support**: Containerized deployment with Docker
- **Graceful Shutdown**: Proper cleanup of routing rules on service termination

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Router 1      │    │   Router 2      │    │   Router N      │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Router Sync │ │    │ │ Router Sync │ │    │ │ Router Sync │ │
│ │   Service   │ │    │ │   Service   │ │    │ │   Service   │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   NATS.io       │
                    │   KV Store      │
                    │   (Source of    │
                    │    Truth)       │
                    └─────────────────┘
```

## Prerequisites

- Go 1.21 or later
- Linux system with root privileges (for netlink operations)
- NATS.io server (with JetStream enabled)
- Network interfaces configured
- Routing tables for each provider already configured

## Quick Start

### Using Docker (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/router-sync.git
   cd router-sync
   ```

2. **Build and run with Docker**
   ```bash
   # Build the Docker image
   make docker-build
   
   # Run the container
   make docker-run
   ```

### Manual Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/router-sync.git
   cd router-sync
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Build the application**
   ```bash
   make build
   ```

4. **Install Swagger documentation generator**
   ```bash
   make install-tools
   ```

5. **Generate API documentation**
   ```bash
   make docs
   ```

## Configuration

Create a `config.yaml` file in the same directory as the binary:

```yaml
# Router Sync Configuration

# Log level (debug, info, warn, error)
log_level: info

# NATS configuration
nats:
  urls:
    - "nats://localhost:4222"
  username: "your-username"  # Optional
  password: "your-password"  # Optional
  token: ""                  # Alternative to username/password
  cluster_id: "router-sync-cluster"
  client_id: "router-sync-client"

# API server configuration
api:
  address: ":8081"  # Default port is 8081

# Synchronization configuration
sync:
  interval: 30s
```

### NATS Configuration

- **urls**: List of NATS server URLs
- **username/password**: Authentication credentials (optional)
- **token**: Alternative authentication method (optional)
- **cluster_id**: NATS cluster identifier
- **client_id**: Unique client identifier

## Usage

### 1. Start the service

```bash
# Using the built binary
sudo ./build/router-sync -config config.yaml

# Using Docker
make docker-run

# Using Makefile
make run
```

**Note**: Root privileges are required for netlink operations when running directly on the host.

### 2. Access the API

The service exposes a REST API on the configured port (default: 8081).

#### API Endpoints

- **Health Check**: `GET /health`
- **API Documentation**: `GET /swagger/*`
- **Prometheus Metrics**: `GET /metrics`

#### Provider Management

- **List Providers**: `GET /api/v1/providers`
- **Create Provider**: `POST /api/v1/providers`
- **Get Provider**: `GET /api/v1/providers/{id}`
- **Update Provider**: `PUT /api/v1/providers/{id}`
- **Delete Provider**: `DELETE /api/v1/providers/{id}`

#### Policy Management

- **List Policies**: `GET /api/v1/policies`
- **Create Policy**: `POST /api/v1/policies`
- **Get Policy**: `GET /api/v1/policies/{id}`
- **Update Policy**: `PUT /api/v1/policies/{id}`
- **Delete Policy**: `DELETE /api/v1/policies/{id}`

**Note**: For CIDR-based policy IDs, use underscore (`_`) instead of slash (`/`) in the URL path. For example:
- Use `192.168.2.0_25` in the URL for policy ID `192.168.2.0/25`
- Use `10.0.0.0_24` in the URL for policy ID `10.0.0.0/24`

#### System Operations

- **Trigger Sync**: `POST /api/v1/sync`
- **Get Stats**: `GET /api/v1/stats`

#### API Request Formats

**Create Provider Request:**
```json
{
  "name": "Provider Name",
  "interface": "eth0",
  "table_id": 100,
  "gateway": "192.168.1.1",
  "description": "Provider description"
}
```

**Create Policy Request:**
```json
{
  "name": "Policy Name",
  "source_ip": "192.168.1.100",
  "provider_id": "Provider Name",
  "priority": 100,
  "description": "Policy description",
  "enabled": true
}
```

**Update Provider Request:**
```json
{
  "name": "Updated Provider Name",
  "interface": "eth1",
  "table_id": 101,
  "gateway": "192.168.1.2",
  "description": "Updated description"
}
```

**Update Policy Request:**
```json
{
  "name": "Updated Policy Name",
  "source_ip": "192.168.1.101",
  "provider_id": "Provider Name",
  "description": "Updated description",
  "enabled": true
}
```

#### Example API Calls

**Create a policy with single IP:**
```bash
curl -X POST http://localhost:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Single IP Policy",
    "source_ip": "192.168.2.24",
    "provider_id": "Starlink",
    "description": "Route single IP through Starlink",
    "enabled": true
  }'
```

**Create a policy with CIDR range:**
```bash
curl -X POST http://localhost:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CIDR Policy",
    "source_ip": "192.168.2.0/25",
    "provider_id": "Tuenti",
    "description": "Route CIDR range through Tuenti",
    "enabled": true
  }'
```

**Update a policy with CIDR (note the underscore in URL):**
```bash
curl -X PUT http://localhost:8081/api/v1/policies/192.168.2.0_25 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated CIDR Policy",
    "source_ip": "192.168.2.0/25",
    "provider_id": "Starlink",
    "description": "Updated description",
    "enabled": true
  }'
```

**Get a policy with CIDR:**
```bash
curl http://localhost:8081/api/v1/policies/192.168.2.0_25
```

**Delete a policy with CIDR:**
```bash
curl -X DELETE http://localhost:8081/api/v1/policies/192.168.2.0_25
```

### 3. Complete Usage Examples

#### Create an Internet Provider

```bash
curl -X POST http://localhost:8081/api/v1/providers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Telecom",
    "interface": "eth0",
    "table_id": 100,
    "gateway": "192.168.1.1",
    "description": "Primary internet connection"
  }'
```

Response:
```json
{
  "id": "Telecom",
  "name": "Telecom",
  "interface": "eth0",
  "table_id": 100,
  "gateway": "192.168.1.1",
  "description": "Primary internet connection",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

#### Create a Routing Policy

```bash
curl -X POST http://localhost:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Home Network",
    "source_ip": "192.168.1.100",
    "provider_id": "Telecom",
    "priority": 100,
    "description": "Route home network through primary provider",
    "enabled": true
  }'
```

Response:
```json
{
  "id": "192.168.1.100",
  "name": "Home Network",
  "provider_id": "Telecom",
  "priority": 100,
  "description": "Route home network through primary provider",
  "enabled": true,
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

#### Create a Policy with CIDR Notation

```bash
curl -X POST http://localhost:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Office Network",
    "source_ip": "192.168.2.0/24",
    "provider_id": "Telecom",
    "priority": 200,
    "description": "Route office network through primary provider",
    "enabled": true
  }'
```

#### List All Providers

```bash
curl -X GET http://localhost:8081/api/v1/providers
```

#### List All Policies

```bash
curl -X GET http://localhost:8081/api/v1/policies
```

#### Get a Specific Provider

```bash
curl -X GET http://localhost:8081/api/v1/providers/Telecom
```

#### Get a Specific Policy

```bash
curl -X GET http://localhost:8081/api/v1/policies/192.168.1.100
```

#### Update a Provider

```bash
curl -X PUT http://localhost:8081/api/v1/providers/Telecom \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Telecom-Updated",
    "interface": "eth1",
    "table_id": 101,
    "gateway": "192.168.1.2",
    "description": "Updated primary internet connection"
  }'
```

#### Update a Policy

```bash
curl -X PUT http://localhost:8081/api/v1/policies/192.168.1.100 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Home Network Updated",
    "source_ip": "192.168.1.101",
    "provider_id": "Telecom",
    "priority": 150,
    "description": "Updated home network routing",
    "enabled": false
  }'
```

#### Delete a Provider

```bash
curl -X DELETE http://localhost:8081/api/v1/providers/Telecom
```

#### Delete a Policy

```bash
curl -X DELETE http://localhost:8081/api/v1/policies/192.168.1.100
```

#### Trigger Manual Sync

```bash
curl -X POST http://localhost:8081/api/v1/sync
```

#### Get System Statistics

```bash
curl -X GET http://localhost:8081/api/v1/stats
```

## Data Models

### InternetProvider

```json
{
  "id": "string",
  "name": "string",
  "interface": "string",
  "table_id": 100,
  "gateway": "192.168.1.1",
  "description": "string",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

### RoutingPolicy

```json
{
  "id": "192.168.1.100",
  "name": "Home Network",
  "provider_id": "Telecom",
  "priority": 100,
  "description": "Route home network through primary provider",
  "enabled": true,
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:00:00Z"
}
```

**Note:** The `source_ip` field from the request is used as the policy ID for routing. The `source_ip` must be a valid IP address or CIDR notation (e.g., "192.168.1.100" or "192.168.1.0/24").

## Monitoring

### Prometheus Metrics

The service exposes the following Prometheus metrics:

- `http_requests_total`: Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds`: HTTP request duration
- `providers_total`: Total number of internet providers
- `policies_total`: Total number of routing policies

### Health Check

```bash
curl http://localhost:8081/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2023-01-01T00:00:00Z",
  "service": "router-sync"
}
```

## Development

### Project Structure

```
router-sync/
├── main.go                 # Application entry point
├── config.yaml            # Configuration file
├── Dockerfile             # Docker container definition
├── Makefile               # Build and development tasks
├── go.mod                 # Go module file
├── go.sum                 # Go module checksums
├── README.md              # This file
└── internal/
    ├── api/               # API server and handlers
    ├── config/            # Configuration management
    ├── models/            # Data models
    ├── nats/              # NATS client
    ├── router/            # Router management (netlink)
    └── sync/              # Synchronization service
```

### Development Commands

```bash
# Build the application
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt

# Run all checks
make check

# Generate API documentation
make docs

# Run locally
make run

# Run with debug logging
make run-debug

# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

### Adding New Features

1. Create feature branch
2. Implement changes
3. Add tests
4. Update documentation
5. Submit pull request

## Testing

Run the test suite:

```bash
make test
```

Run tests with coverage:

```bash
make test-coverage
```

Run tests with race detection:

```bash
make test-race
```

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure the service runs with root privileges when running directly on host
2. **NATS Connection Failed**: Check NATS server configuration and network connectivity
3. **Interface Not Found**: Verify network interface names exist on the system
4. **Invalid Gateway**: Ensure gateway IP addresses are valid and reachable
5. **Port Already in Use**: Check if port 8081 is available or change it in config.yaml

### Logs

The service uses structured logging with different levels:

- **DEBUG**: Detailed debugging information
- **INFO**: General information about operations
- **WARN**: Warning messages for non-critical issues
- **ERROR**: Error messages for critical issues

### Debug Mode

Enable debug logging by setting `log_level: debug` in the configuration file.

## Security Considerations

- Run the service with minimal required privileges
- Use NATS authentication for production deployments
- Secure the API endpoints in production environments
- Regularly update dependencies for security patches
- Consider using Docker for better isolation

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`make check`)
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:

- Create an issue on GitHub
- Check the documentation
- Review the troubleshooting section

## Roadmap

- [x] Docker containerization
- [x] Graceful shutdown with cleanup
- [x] Comprehensive Makefile
- [ ] Web UI for configuration management
- [ ] Support for IPv6
- [ ] Advanced routing policies (load balancing, failover)
- [ ] Integration with monitoring systems
- [ ] Kubernetes deployment manifests
- [ ] Configuration validation
- [ ] Backup and restore functionality 