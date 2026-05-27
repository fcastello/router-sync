# Router Sync: Simplifying Multi-ISP Routing in High Availability Home Networks

## Introduction

In today's connected world, having reliable internet connectivity is crucial. Many households and small businesses are turning to multi-ISP setups to ensure redundancy and optimize bandwidth usage. However, managing routing across multiple internet service providers can be complex, especially when you need to route specific devices or network segments through different ISPs.

This is where **Router Sync** comes in - a powerful, open-source solution that simplifies policy-based routing across multiple routers in a high-availability environment.

## The Problem: Complex Multi-ISP Routing

### My Home Network Setup

I run a sophisticated home network with two Linux routers configured for high availability, connected to multiple internet service providers. Here's what my setup looks like:

```
                    ┌─────────────────┐    ┌─────────────────┐
                    │   Router 1      │    │   Router 2      │
                    │   (Primary)     │    │   (Secondary)   │
                    │                 │    │                 │
                    │ ┌─────────────┐ │    │ ┌─────────────┐ │
                    │ │ Router Sync │ │    │ │ Router Sync │ │
                    │ │   Service   │ │    │ │   Service   │ │
                    │ └─────────────┘ │    │ └─────────────┘ │
                    └─────────────────┘    └─────────────────┘
                             │                       │
                    ┌────────┴────────┐    ┌────────┴────────┐
                    │                 │    │                 │
                    │  ISP 1 (Fiber)  │    │  ISP 2 (Cable)  │
                    │                 │    │                 │
                    └─────────────────┘    └─────────────────┘
                             │                       │
                    ┌────────┴────────┐    ┌────────┴────────┐
                    │                 │    │                 │
                    │  ISP 3 (4G)     │    │  ISP 4 (Starlink)│
                    │                 │    │                 │
                    └─────────────────┘    └─────────────────┘
```

### The Challenges I Faced

Before implementing Router Sync, I encountered several challenges:

1. **Manual Configuration**: Every time I wanted to route a device through a different ISP, I had to manually configure routing rules on both routers
2. **Configuration Drift**: Keeping routing policies synchronized between routers was error-prone
3. **Complex Failover**: When one router failed, I had to manually reconfigure routing on the backup router
4. **No Centralized Management**: There was no single source of truth for routing policies
5. **Time-Consuming Changes**: Adding new devices or changing routing policies required manual intervention on multiple systems

## The Solution: Router Sync

Router Sync is a Go-based service that manages internet providers and routing policies using NATS.io as the source of truth. It enables policy-based routing across multiple routers in a LAN environment with automatic synchronization.

### Key Features

- **Internet Provider Management**: Add, remove, and manage internet service providers with their associated network interfaces and routing tables
- **Policy-Based Routing**: Create routing policies based on source IP addresses (single IP or CIDR notation)
- **NATS.io Integration**: Uses NATS.io key-value store for persistent configuration storage
- **Real-time Synchronization**: Automatic synchronization between NATS KV store and router configuration
- **REST API**: Full CRUD operations for providers and policies
- **High Availability**: Works seamlessly across multiple routers
- **Graceful Shutdown**: Proper cleanup of routing rules on service termination

## Real-World Usage Scenarios

### Scenario 1: Device-Specific Routing

I have several IoT devices that need to use different ISPs for various reasons:

```bash
# Route my security cameras through the most stable ISP (Fiber)
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Security Cameras",
    "source_ip": "192.168.1.100",
    "provider_id": "Fiber-ISP",
    "description": "Route security cameras through fiber for stability",
    "enabled": true
  }'

# Route my gaming PC through the lowest latency ISP (Cable)
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gaming PC",
    "source_ip": "192.168.1.50",
    "provider_id": "Cable-ISP",
    "description": "Route gaming PC through cable for low latency",
    "enabled": true
  }'

# Route my work laptop through the backup ISP (4G) for redundancy
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Work Laptop",
    "source_ip": "192.168.1.25",
    "provider_id": "4G-ISP",
    "description": "Route work laptop through 4G for redundancy",
    "enabled": true
  }'
```

### Scenario 2: Subnet-Based Routing

I also use CIDR notation to route entire network segments:

```bash
# Route all devices in the guest network through Starlink
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Guest Network",
    "source_ip": "192.168.2.0/24",
    "provider_id": "Starlink-ISP",
    "description": "Route guest network through Starlink",
    "enabled": true
  }'

# Route IoT devices subnet through the most stable connection
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "IoT Network",
    "source_ip": "192.168.3.0/25",
    "provider_id": "Fiber-ISP",
    "description": "Route IoT devices through fiber for stability",
    "enabled": true
  }'
```

### Scenario 3: Dynamic ISP Switching

One of the most powerful features is the ability to quickly switch ISPs for specific devices:

```bash
# Switch my streaming device from Cable to Fiber for better bandwidth
curl -X PUT http://router1:8081/api/v1/policies/192.168.1.75 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Streaming Device",
    "source_ip": "192.168.1.75",
    "provider_id": "Fiber-ISP",
    "description": "Streaming device now using fiber for better bandwidth",
    "enabled": true
  }'
```

## High Availability Benefits

### Automatic Failover

When Router 1 fails, Router 2 automatically takes over with the same routing policies:

1. **NATS as Source of Truth**: All routing policies are stored in NATS.io KV store
2. **Automatic Sync**: Router 2's Router Sync service automatically syncs all policies
3. **Zero Configuration**: No manual intervention required during failover
4. **Consistent State**: Both routers maintain identical routing configurations

### Configuration Synchronization

```bash
# Check current policies on Router 1
curl http://router1:8081/api/v1/policies

# Check current policies on Router 2 (should be identical)
curl http://router2:8081/api/v1/policies

# Both return the same policies, ensuring consistency
```

## Installation and Setup

### Quick Start with Docker

```bash
# Clone the repository
git clone https://github.com/yourusername/router-sync.git
cd router-sync

# Build and run with Docker
make docker-build
make docker-run
```

### Linux Installation (Systemd Service)

For production deployments on Linux routers:

```bash
# Download the latest release
wget https://github.com/yourusername/router-sync/releases/latest/download/router-sync-v<VERSION>-linux-amd64.tar.gz

# Extract and install
tar -xzf router-sync-v<VERSION>-linux-amd64.tar.gz
cd router-sync-v<VERSION>-linux-amd64
sudo ./install.sh
```

The installation script automatically:
- Creates a dedicated system user
- Installs the binary to `/usr/local/bin/`
- Creates configuration directory at `/etc/router-sync/`
- Installs and enables the systemd service
- Starts the service automatically

## Configuration Example

Here's my actual configuration file:

```yaml
# Router Sync Configuration
log_level: info

# NATS configuration
nats:
  urls:
    - "nats://192.168.1.10:4222"  # My NATS server
  username: "router-sync"
  password: "secure-password"
  cluster_id: "home-network"
  client_id: "router-sync-router1"

# API server configuration
api:
  address: ":8081"

# Synchronization configuration
sync:
  interval: 30s
```

## Monitoring and Management

### API Endpoints

Router Sync provides a comprehensive REST API:

```bash
# Health check
curl http://router1:8081/health

# List all providers
curl http://router1:8081/api/v1/providers

# List all policies
curl http://router1:8081/api/v1/policies

# Get system statistics
curl http://router1:8081/api/v1/stats

# Trigger manual sync
curl -X POST http://router1:8081/api/v1/sync
```

### Prometheus Metrics

The service exposes comprehensive metrics for monitoring:

```bash
# View metrics
curl http://router1:8081/metrics
```

Key metrics include:
- `http_requests_total`: Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds`: HTTP request duration
- `providers_total`: Total number of internet providers
- `policies_total`: Total number of routing policies

## Real-World Benefits

### Before Router Sync

- **Manual Configuration**: 15-20 minutes to add a new device to a different ISP
- **Error-Prone**: Frequent configuration mistakes leading to connectivity issues
- **No Consistency**: Routers often had different routing states
- **Complex Failover**: Manual intervention required during router failures
- **No Monitoring**: No visibility into routing policy status

### After Router Sync

- **Instant Changes**: New routing policies applied in seconds via API
- **Zero Errors**: Automated configuration eliminates human error
- **Perfect Consistency**: NATS ensures both routers are always in sync
- **Automatic Failover**: Seamless failover with no manual intervention
- **Full Monitoring**: Complete visibility into routing policy status and health

## Advanced Use Cases

### Load Balancing

While Router Sync doesn't provide load balancing directly, you can implement it by creating multiple policies for the same device and enabling/disabling them based on load:

```bash
# Enable backup ISP for a device when primary is congested
curl -X PUT http://router1:8081/api/v1/policies/192.168.1.100 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Load Balanced Device",
    "source_ip": "192.168.1.100",
    "provider_id": "Backup-ISP",
    "description": "Using backup ISP due to primary congestion",
    "enabled": true
  }'
```

### Geographic Routing

Route devices through ISPs that provide better connectivity to specific geographic regions:

```bash
# Route VPN server through ISP with better international routing
curl -X POST http://router1:8081/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "VPN Server",
    "source_ip": "192.168.1.200",
    "provider_id": "International-ISP",
    "description": "VPN server using ISP with better international routing",
    "enabled": true
  }'
```

## Troubleshooting

### Common Issues and Solutions

1. **Permission Denied**: Ensure the service runs with root privileges for netlink operations
2. **NATS Connection Failed**: Check NATS server configuration and network connectivity
3. **Interface Not Found**: Verify network interface names exist on the system
4. **Invalid Gateway**: Ensure gateway IP addresses are valid and reachable

### Debug Mode

Enable debug logging for troubleshooting:

```yaml
log_level: debug
```

### Service Management

```bash
# Check service status
sudo systemctl status router-sync

# View logs
sudo journalctl -u router-sync -f

# Restart service
sudo systemctl restart router-sync
```

## Conclusion

Router Sync has transformed my multi-ISP home network from a complex, error-prone setup into a reliable, easily manageable system. The ability to instantly route any device or network segment through any ISP with a simple API call has made network management effortless.

Key benefits I've experienced:

- **Reliability**: Zero downtime during router failovers
- **Flexibility**: Easy switching of devices between ISPs
- **Simplicity**: Complex routing decisions reduced to simple API calls
- **Consistency**: Both routers always maintain identical configurations
- **Monitoring**: Complete visibility into routing policy status

Whether you're running a home network with multiple ISPs or managing a small business network, Router Sync provides the tools you need to implement sophisticated policy-based routing without the complexity.

The project is open-source and actively maintained, with comprehensive documentation and a growing community. If you're struggling with multi-ISP routing management, I highly recommend giving Router Sync a try.

## Resources

- **GitHub Repository**: [https://github.com/yourusername/router-sync](https://github.com/yourusername/router-sync)
- **Documentation**: [https://github.com/yourusername/router-sync/blob/main/README.md](https://github.com/yourusername/router-sync/blob/main/README.md)
- **Architecture Guide**: [https://github.com/yourusername/router-sync/blob/main/ARCHITECTURE.md](https://github.com/yourusername/router-sync/blob/main/ARCHITECTURE.md)
- **Installation Guide**: [https://github.com/yourusername/router-sync/blob/main/scripts/README.md](https://github.com/yourusername/router-sync/blob/main/scripts/README.md)

---

*Router Sync: Making multi-ISP routing management simple and reliable.*
