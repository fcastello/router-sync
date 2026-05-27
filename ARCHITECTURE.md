# Router Sync Architecture

## Overview

Router Sync is a Go-based service that manages internet providers and routing policies using NATS.io as the source of truth. It enables policy-based routing across multiple routers in a LAN environment by synchronizing configuration between NATS KV store and Linux routing tables.

## System Architecture

### High-Level Architecture

```mermaid
graph TB
    subgraph "Router 1"
        RS1[Router Sync Service]
        RT1[Routing Tables]
        NL1[Netlink Interface]
    end
    
    subgraph "Router 2"
        RS2[Router Sync Service]
        RT2[Routing Tables]
        NL2[Netlink Interface]
    end
    
    subgraph "Router N"
        RSN[Router Sync Service]
        RTN[Routing Tables]
        NLN[Netlink Interface]
    end
    
    subgraph "NATS Cluster"
        NATS[NATS.io Server]
        KV[Key-Value Store]
        JS[JetStream]
    end
    
    subgraph "API Layer"
        API[REST API Server]
        SWAGGER[Swagger Docs]
        METRICS[Prometheus Metrics]
    end
    
    RS1 --> NATS
    RS2 --> NATS
    RSN --> NATS
    API --> NATS
    
    RS1 --> NL1
    RS2 --> NL2
    RSN --> NLN
    
    NL1 --> RT1
    NL2 --> RT2
    NLN --> RTN
    
    NATS --> KV
    NATS --> JS
    API --> SWAGGER
    API --> METRICS
```

## Component Architecture

### Core Components

```mermaid
graph LR
    subgraph "Main Application"
        MAIN[main.go]
        CONFIG[Config Manager]
    end
    
    subgraph "API Layer"
        SERVER[API Server]
        HANDLERS[HTTP Handlers]
        MIDDLEWARE[Middleware]
    end
    
    subgraph "Sync Layer"
        SYNC[Sync Service]
        WATCHER[Change Watchers]
        CACHE[State Cache]
    end
    
    subgraph "Router Layer"
        MANAGER[Router Manager]
        NETLINK[Netlink Operations]
        IPCMD[IP Commands]
    end
    
    subgraph "Storage Layer"
        NATS[NATS Client]
        KVSTORE[KV Store]
        JETSTREAM[JetStream]
    end
    
    subgraph "Models"
        PROVIDER[InternetProvider]
        POLICY[RoutingPolicy]
    end
    
    MAIN --> CONFIG
    MAIN --> SERVER
    MAIN --> SYNC
    MAIN --> MANAGER
    MAIN --> NATS
    
    SERVER --> HANDLERS
    SERVER --> MIDDLEWARE
    HANDLERS --> NATS
    HANDLERS --> MANAGER
    
    SYNC --> NATS
    SYNC --> MANAGER
    SYNC --> WATCHER
    SYNC --> CACHE
    
    MANAGER --> NETLINK
    MANAGER --> IPCMD
    
    NATS --> KVSTORE
    NATS --> JETSTREAM
    
    HANDLERS --> PROVIDER
    HANDLERS --> POLICY
    SYNC --> PROVIDER
    SYNC --> POLICY
```

## Data Flow Architecture

### Service Startup Sequence

```mermaid
sequenceDiagram
    participant M as Main
    participant C as Config
    participant N as NATS Client
    participant R as Router Manager
    participant S as Sync Service
    participant API as API Server
    
    M->>C: Load configuration
    C-->>M: Return config
    
    M->>N: Initialize NATS connection
    N->>N: Connect to NATS server
    N->>N: Create KV store
    N-->>M: Return NATS client
    
    M->>R: Initialize router manager
    R->>R: Setup netlink interface
    R-->>M: Return router manager
    
    M->>S: Initialize sync service
    S->>S: Setup context and cache
    S-->>M: Return sync service
    
    M->>API: Initialize API server
    API->>API: Setup Gin router
    API->>API: Register middleware
    API->>API: Setup routes
    API-->>M: Return API server
    
    M->>S: Start sync service
    S->>S: Perform initial sync
    S->>S: Start periodic sync
    S->>S: Start watchers
    
    M->>API: Start API server
    API->>API: Listen on port
    
    Note over M: Wait for shutdown signal
```

### Provider Management Sequence

```mermaid
sequenceDiagram
    participant C as Client
    participant API as API Server
    participant H as HTTP Handler
    participant N as NATS Client
    participant KV as KV Store
    participant S as Sync Service
    participant R as Router Manager
    
    C->>API: POST /api/v1/providers
    API->>H: Create provider handler
    H->>H: Validate request
    H->>N: Store provider
    N->>KV: Put provider data
    KV-->>N: Confirm storage
    N-->>H: Return success
    H-->>API: Return response
    API-->>C: 201 Created
    
    Note over S: NATS watcher detects change
    S->>S: Update provider cache
    S->>R: Setup provider routing
    R->>R: Configure routing table
    R-->>S: Confirm setup
```

### Policy Management Sequence

```mermaid
sequenceDiagram
    participant C as Client
    participant API as API Server
    participant H as HTTP Handler
    participant N as NATS Client
    participant KV as KV Store
    participant S as Sync Service
    participant R as Router Manager
    participant NL as Netlink
    
    C->>API: POST /api/v1/policies
    API->>H: Create policy handler
    H->>H: Validate request
    H->>N: Store policy
    N->>KV: Put policy data
    KV-->>N: Confirm storage
    N-->>H: Return success
    H-->>API: Return response
    API-->>C: 201 Created
    
    Note over S: NATS watcher detects change
    S->>S: Update policy cache
    S->>S: Get associated provider
    S->>R: Setup policy routing
    R->>NL: Add routing rule
    NL->>NL: Configure source-based routing
    NL-->>R: Confirm rule addition
    R-->>S: Confirm setup
```

### Synchronization Sequence

```mermaid
sequenceDiagram
    participant S as Sync Service
    participant N as NATS Client
    participant KV as KV Store
    participant R as Router Manager
    participant NL as Netlink
    participant C as Cache
    
    S->>S: Start periodic sync
    S->>N: List all providers
    N->>KV: Get provider keys
    KV-->>N: Return provider data
    N-->>S: Return providers
    
    S->>N: List all policies
    N->>KV: Get policy keys
    KV-->>N: Return policy data
    N-->>S: Return policies
    
    S->>C: Update provider cache
    S->>C: Update policy cache
    
    S->>R: Sync policies with providers
    R->>R: Clear existing rules
    R->>NL: Remove stale rules
    R->>NL: Add new routing rules
    R->>NL: Validate rule consistency
    NL-->>R: Confirm operations
    R-->>S: Confirm sync completion
```

## Object Model

### Core Data Models

```mermaid
classDiagram
    class InternetProvider {
        +string ID
        +string Name
        +string Interface
        +int TableID
        +string Gateway
        +string Description
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +Validate() error
        +ToJSON() ([]byte, error)
        +FromJSON([]byte) error
    }
    
    class RoutingPolicy {
        +string ID
        +string Name
        +string ProviderID
        +string Description
        +bool Enabled
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +Validate() error
        +ToJSON() ([]byte, error)
        +FromJSON([]byte) error
    }
    
    class Config {
        +logrus.Level LogLevel
        +NATSConfig NATS
        +APIConfig API
        +SyncConfig Sync
        +Load(string) (*Config, error)
    }
    
    class NATSConfig {
        +[]string URLs
        +string Username
        +string Password
        +string Token
        +string ClusterID
        +string ClientID
    }
    
    class APIConfig {
        +string Address
    }
    
    class SyncConfig {
        +time.Duration Interval
    }
    
    RoutingPolicy --> InternetProvider : references
    Config --> NATSConfig : contains
    Config --> APIConfig : contains
    Config --> SyncConfig : contains
```

### Service Architecture

```mermaid
classDiagram
    class Server {
        +config.APIConfig config
        +nats.NATSClient natsClient
        +*router.Manager routerManager
        +*sync.Service syncService
        +*http.Server server
        +*prometheus.CounterVec httpRequestsTotal
        +*prometheus.HistogramVec httpRequestDuration
        +prometheus.Gauge providersTotal
        +prometheus.Gauge policiesTotal
        +string version
        +string buildTime
        +string gitCommit
        +NewServer() *Server
        +Start() error
        +Shutdown(context.Context) error
        +metricsMiddleware() gin.HandlerFunc
        +urlDecodeMiddleware() gin.HandlerFunc
    }
    
    class Service {
        +*nats.Client natsClient
        +*router.Manager routerManager
        +config.SyncConfig config
        +context.Context ctx
        +context.CancelFunc cancel
        +sync.WaitGroup wg
        +map[string]*models.InternetProvider providers
        +map[string]*models.RoutingPolicy policies
        +sync.RWMutex cacheMu
        +NewService() *Service
        +Start() error
        +Stop() error
        +performFullSync() error
        +watchProviders() error
        +watchPolicies() error
        +GetStats() map[string]interface{}
    }
    
    class Manager {
        +sync.RWMutex mu
        +NewManager() (*Manager, error)
        +SetupProvider(*models.InternetProvider) error
        +RemoveProvider(*models.InternetProvider) error
        +SetupPolicy(*models.RoutingPolicy, *models.InternetProvider) error
        +RemovePolicy(*models.RoutingPolicy, *models.InternetProvider) error
        +SyncProviders([]*models.InternetProvider) error
        +SyncPolicies([]*models.RoutingPolicy, []*models.InternetProvider) error
        +GetRoutingStats() (map[string]interface{}, error)
        +CleanupAllRules() error
    }
    
    class Client {
        +*nats.Conn conn
        +nats.JetStreamContext js
        +nats.KeyValue kv
        +NewClient(config.NATSConfig) (*Client, error)
        +Close()
        +StoreProvider(*models.InternetProvider) error
        +GetProvider(string) (*models.InternetProvider, error)
        +ListProviders() ([]*models.InternetProvider, error)
        +DeleteProvider(string) error
        +StorePolicy(*models.RoutingPolicy) error
        +GetPolicy(string) (*models.RoutingPolicy, error)
        +ListPolicies() ([]*models.RoutingPolicy, error)
        +DeletePolicy(string) error
        +WatchProviders(context.Context, func) error
        +WatchPolicies(context.Context, func) error
    }
    
    Server --> Service : uses
    Server --> Manager : uses
    Server --> Client : uses
    Service --> Manager : uses
    Service --> Client : uses
```

## API Architecture

### REST API Endpoints

```mermaid
graph TD
    subgraph "API Endpoints"
        HEALTH[GET /health]
        METRICS[GET /metrics]
        SWAGGER[GET /swagger/*]
        
        subgraph "Provider Management"
            LIST_PROVIDERS[GET /api/v1/providers]
            CREATE_PROVIDER[POST /api/v1/providers]
            GET_PROVIDER[GET /api/v1/providers/:id]
            UPDATE_PROVIDER[PUT /api/v1/providers/:id]
            DELETE_PROVIDER[DELETE /api/v1/providers/:id]
        end
        
        subgraph "Policy Management"
            LIST_POLICIES[GET /api/v1/policies]
            CREATE_POLICY[POST /api/v1/policies]
            GET_POLICY[GET /api/v1/policies/:id]
            UPDATE_POLICY[PUT /api/v1/policies/:id]
            DELETE_POLICY[DELETE /api/v1/policies/:id]
        end
        
        subgraph "System Operations"
            TRIGGER_SYNC[POST /api/v1/sync]
            GET_STATS[GET /api/v1/stats]
        end
    end
    
    HEALTH --> API_SERVER
    METRICS --> API_SERVER
    SWAGGER --> API_SERVER
    LIST_PROVIDERS --> API_SERVER
    CREATE_PROVIDER --> API_SERVER
    GET_PROVIDER --> API_SERVER
    UPDATE_PROVIDER --> API_SERVER
    DELETE_PROVIDER --> API_SERVER
    LIST_POLICIES --> API_SERVER
    CREATE_POLICY --> API_SERVER
    GET_POLICY --> API_SERVER
    UPDATE_POLICY --> API_SERVER
    DELETE_POLICY --> API_SERVER
    TRIGGER_SYNC --> API_SERVER
    GET_STATS --> API_SERVER
```

## Storage Architecture

### NATS Key-Value Store Structure

```mermaid
graph TD
    subgraph "NATS KV Store"
        subgraph "Providers"
            P1[providers.provider1]
            P2[providers.provider2]
            P3[providers.provider3]
        end
        
        subgraph "Policies"
            POL1[policies.192.168.1.100]
            POL2[policies.192.168.2.0_24]
            POL3[policies.10.0.0.0_16]
        end
    end
    
    subgraph "Data Format"
        PROVIDER_JSON[{"id":"provider1","name":"ISP1","interface":"eth0","table_id":100,"gateway":"192.168.1.1"}]
        POLICY_JSON[{"id":"192.168.1.100","name":"Policy1","provider_id":"provider1","enabled":true}]
    end
    
    P1 --> PROVIDER_JSON
    P2 --> PROVIDER_JSON
    P3 --> PROVIDER_JSON
    POL1 --> POLICY_JSON
    POL2 --> POLICY_JSON
    POL3 --> POLICY_JSON
```

## Routing Architecture

### Linux Routing Table Structure

```mermaid
graph TD
    subgraph "Linux Routing Tables"
        subgraph "Main Table (Table 254)"
            MAIN_DEFAULT[default via 192.168.1.1 dev eth0]
            MAIN_LOCAL[192.168.1.0/24 dev eth0]
        end
        
        subgraph "Provider Tables"
            TABLE_100[Table 100 - ISP1]
            TABLE_101[Table 101 - ISP2]
            TABLE_102[Table 102 - ISP3]
        end
        
        subgraph "Routing Rules"
            RULE_1[from 192.168.1.100 lookup 100]
            RULE_2[from 192.168.2.0/24 lookup 101]
            RULE_3[from 10.0.0.0/16 lookup 102]
        end
    end
    
    RULE_1 --> TABLE_100
    RULE_2 --> TABLE_101
    RULE_3 --> TABLE_102
```

## Deployment Architecture

### Docker Deployment

```mermaid
graph TB
    subgraph "Docker Environment"
        subgraph "Router Host"
            CONTAINER[Router Sync Container]
            NETWORK[Host Network]
            ROUTING[Linux Routing Tables]
        end
        
        subgraph "NATS Server"
            NATS_SERVER[NATS.io Server]
            JETSTREAM[JetStream]
            KV_STORE[Key-Value Store]
        end
        
        subgraph "External Services"
            PROMETHEUS[Prometheus]
            GRAFANA[Grafana]
            LOGS[Log Aggregation]
        end
    end
    
    CONTAINER --> NETWORK
    NETWORK --> ROUTING
    CONTAINER --> NATS_SERVER
    NATS_SERVER --> JETSTREAM
    JETSTREAM --> KV_STORE
    CONTAINER --> PROMETHEUS
    PROMETHEUS --> GRAFANA
    CONTAINER --> LOGS
```

### Systemd Service Deployment

```mermaid
graph TD
    subgraph "Linux System"
        subgraph "Systemd"
            SERVICE[router-sync.service]
            USER[router-sync user]
        end
        
        subgraph "File System"
            BINARY[/usr/local/bin/router-sync]
            CONFIG[/etc/router-sync/config.yaml]
            LOGS[/var/log/router-sync]
        end
        
        subgraph "Network"
            NETLINK[Netlink Interface]
            ROUTING[Routing Tables]
        end
    end
    
    SERVICE --> USER
    USER --> BINARY
    BINARY --> CONFIG
    BINARY --> LOGS
    BINARY --> NETLINK
    NETLINK --> ROUTING
```

## Monitoring Architecture

### Metrics and Observability

```mermaid
graph TD
    subgraph "Application Metrics"
        HTTP_METRICS[http_requests_total]
        DURATION_METRICS[http_request_duration_seconds]
        PROVIDER_METRICS[providers_total]
        POLICY_METRICS[policies_total]
    end
    
    subgraph "System Metrics"
        SYNC_STATS[Sync Statistics]
        ROUTING_STATS[Routing Statistics]
        NATS_STATS[NATS Connection Stats]
    end
    
    subgraph "Monitoring Stack"
        PROMETHEUS[Prometheus]
        GRAFANA[Grafana]
        ALERTMANAGER[AlertManager]
    end
    
    HTTP_METRICS --> PROMETHEUS
    DURATION_METRICS --> PROMETHEUS
    PROVIDER_METRICS --> PROMETHEUS
    POLICY_METRICS --> PROMETHEUS
    SYNC_STATS --> PROMETHEUS
    ROUTING_STATS --> PROMETHEUS
    NATS_STATS --> PROMETHEUS
    
    PROMETHEUS --> GRAFANA
    PROMETHEUS --> ALERTMANAGER
```

## Security Architecture

### Authentication and Authorization

```mermaid
graph TD
    subgraph "NATS Authentication"
        USERNAME[Username/Password]
        TOKEN[Token Authentication]
        TLS[TLS Encryption]
    end
    
    subgraph "API Security"
        HTTPS[HTTPS/TLS]
        CORS[CORS Headers]
        RATE_LIMIT[Rate Limiting]
    end
    
    subgraph "System Security"
        ROOT_PRIV[Root Privileges]
        NETLINK_ACCESS[Netlink Access]
        FILE_PERMS[File Permissions]
    end
    
    USERNAME --> NATS_CLIENT
    TOKEN --> NATS_CLIENT
    TLS --> NATS_CLIENT
    HTTPS --> API_SERVER
    CORS --> API_SERVER
    RATE_LIMIT --> API_SERVER
    ROOT_PRIV --> ROUTER_MANAGER
    NETLINK_ACCESS --> ROUTER_MANAGER
    FILE_PERMS --> CONFIG_LOADER
```

## Error Handling Architecture

### Error Flow and Recovery

```mermaid
graph TD
    subgraph "Error Types"
        CONFIG_ERROR[Configuration Errors]
        NATS_ERROR[NATS Connection Errors]
        ROUTING_ERROR[Routing Operation Errors]
        VALIDATION_ERROR[Validation Errors]
        SYSTEM_ERROR[System Errors]
    end
    
    subgraph "Error Handling"
        LOGGING[Structured Logging]
        METRICS[Error Metrics]
        RECOVERY[Automatic Recovery]
        GRACEFUL_DEGRADE[Graceful Degradation]
    end
    
    subgraph "Recovery Mechanisms"
        RETRY[Exponential Backoff]
        CIRCUIT_BREAKER[Circuit Breaker]
        FALLBACK[Fallback Behavior]
        CLEANUP[Resource Cleanup]
    end
    
    CONFIG_ERROR --> LOGGING
    NATS_ERROR --> LOGGING
    ROUTING_ERROR --> LOGGING
    VALIDATION_ERROR --> LOGGING
    SYSTEM_ERROR --> LOGGING
    
    LOGGING --> METRICS
    LOGGING --> RECOVERY
    RECOVERY --> GRACEFUL_DEGRADE
    
    RECOVERY --> RETRY
    RECOVERY --> CIRCUIT_BREAKER
    RECOVERY --> FALLBACK
    RECOVERY --> CLEANUP
```

## Performance Architecture

### Scalability and Performance

```mermaid
graph TD
    subgraph "Performance Factors"
        CONCURRENT_SYNC[Concurrent Synchronization]
        CACHE_PERFORMANCE[Cache Performance]
        NETLINK_EFFICIENCY[Netlink Efficiency]
        NATS_THROUGHPUT[NATS Throughput]
    end
    
    subgraph "Optimization Strategies"
        BATCH_OPERATIONS[Batch Operations]
        CONNECTION_POOLING[Connection Pooling]
        MEMORY_MANAGEMENT[Memory Management]
        ASYNC_PROCESSING[Async Processing]
    end
    
    subgraph "Resource Management"
        CPU_USAGE[CPU Usage]
        MEMORY_USAGE[Memory Usage]
        NETWORK_IO[Network I/O]
        DISK_IO[Disk I/O]
    end
    
    CONCURRENT_SYNC --> BATCH_OPERATIONS
    CACHE_PERFORMANCE --> MEMORY_MANAGEMENT
    NETLINK_EFFICIENCY --> ASYNC_PROCESSING
    NATS_THROUGHPUT --> CONNECTION_POOLING
    
    BATCH_OPERATIONS --> CPU_USAGE
    MEMORY_MANAGEMENT --> MEMORY_USAGE
    ASYNC_PROCESSING --> NETWORK_IO
    CONNECTION_POOLING --> DISK_IO
```

## Testing Architecture

### Testing Strategy

```mermaid
graph TD
    subgraph "Test Types"
        UNIT_TESTS[Unit Tests]
        INTEGRATION_TESTS[Integration Tests]
        E2E_TESTS[End-to-End Tests]
        PERFORMANCE_TESTS[Performance Tests]
    end
    
    subgraph "Test Components"
        MOCK_NATS[Mock NATS Client]
        MOCK_ROUTER[Mock Router Manager]
        TEST_CONFIG[Test Configuration]
        TEST_DATA[Test Data Sets]
    end
    
    subgraph "Test Infrastructure"
        TEST_CONTAINERS[Test Containers]
        CI_CD[CI/CD Pipeline]
        TEST_REPORTS[Test Reports]
        COVERAGE[Code Coverage]
    end
    
    UNIT_TESTS --> MOCK_NATS
    UNIT_TESTS --> MOCK_ROUTER
    INTEGRATION_TESTS --> TEST_CONFIG
    INTEGRATION_TESTS --> TEST_DATA
    E2E_TESTS --> TEST_CONTAINERS
    PERFORMANCE_TESTS --> TEST_CONTAINERS
    
    UNIT_TESTS --> CI_CD
    INTEGRATION_TESTS --> CI_CD
    E2E_TESTS --> CI_CD
    PERFORMANCE_TESTS --> CI_CD
    
    CI_CD --> TEST_REPORTS
    CI_CD --> COVERAGE
```

This architecture documentation provides a comprehensive view of the Router Sync service, including its components, data flow, deployment strategies, and operational considerations. The diagrams help visualize the relationships between different components and the overall system behavior. 