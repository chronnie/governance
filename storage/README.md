# Governance Storage

This package provides pluggable storage backends for the governance library. You can choose between in-memory storage (default) or persistent database storage (MySQL, PostgreSQL, MongoDB).

## Storage Interface

All storage implementations conform to the `RegistryStore` interface:

```go
type RegistryStore interface {
    // Service operations
    SaveService(ctx context.Context, service *models.ServiceInfo) error
    GetService(ctx context.Context, key string) (*models.ServiceInfo, error)
    GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error)
    GetAllServices(ctx context.Context) ([]*models.ServiceInfo, error)
    DeleteService(ctx context.Context, key string) error
    UpdateHealthStatus(ctx context.Context, key string, status models.ServiceStatus, timestamp time.Time) error

    // Subscription operations
    AddSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error
    RemoveSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error
    RemoveAllSubscriptions(ctx context.Context, subscriberKey string) error
    GetSubscribers(ctx context.Context, serviceGroup string) ([]string, error)
    GetSubscriberServices(ctx context.Context, serviceGroup string) ([]*models.ServiceInfo, error)

    // Lifecycle operations
    Close() error
    Ping(ctx context.Context) error
}
```

## Available Storage Backends

### 1. In-Memory Storage (Default)

Fast, lock-free in-memory storage. Data is lost when the manager stops.

```go
import (
    "github.com/chronnie/governance/manager"
    "github.com/chronnie/governance/models"
)

// In-memory storage is used by default
mgr := manager.NewManager(config)
```

### 2. MySQL Storage

Persistent storage using MySQL database.

**Setup:**

```sql
CREATE DATABASE governance;
```

**Usage:**

```go
import (
    "github.com/chronnie/governance/manager"
    "github.com/chronnie/governance/storage/mysql"
)

mysqlConfig := mysql.Config{
    Host:     "localhost",
    Port:     3306,
    Database: "governance",
    Username: "root",
    Password: "password",
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
}

store, err := mysql.NewMySQLStore(mysqlConfig)
if err != nil {
    log.Fatal(err)
}

mgr := manager.NewManagerWithStorage(config, store)
```

**Tables Created:**
- `services` - Stores service registration data
- `subscriptions` - Stores pub/sub relationships

### 3. PostgreSQL Storage

Persistent storage using PostgreSQL database.

**Setup:**

```sql
CREATE DATABASE governance;
```

**Usage:**

```go
import (
    "github.com/chronnie/governance/manager"
    "github.com/chronnie/governance/storage/postgres"
)

postgresConfig := postgres.Config{
    Host:     "localhost",
    Port:     5432,
    Database: "governance",
    Username: "postgres",
    Password: "password",
    SSLMode:  "disable", // or "require", "verify-ca", "verify-full"
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
}

store, err := postgres.NewPostgreSQLStore(postgresConfig)
if err != nil {
    log.Fatal(err)
}

mgr := manager.NewManagerWithStorage(config, store)
```

**Tables Created:**
- `services` - Stores service registration data (with JSONB for flexible data)
- `subscriptions` - Stores pub/sub relationships

### 4. MongoDB Storage

Persistent storage using MongoDB.

**Setup:**

MongoDB will automatically create the database and collections.

**Usage:**

```go
import (
    "github.com/chronnie/governance/manager"
    "github.com/chronnie/governance/storage/mongodb"
)

mongoConfig := mongodb.Config{
    URI:            "mongodb://localhost:27017",
    Database:       "governance",
    ConnectTimeout: 10 * time.Second,
    MaxPoolSize: 100,
    MinPoolSize: 10,
}

store, err := mongodb.NewMongoDBStore(mongoConfig)
if err != nil {
    log.Fatal(err)
}

mgr := manager.NewManagerWithStorage(config, store)
```

**Collections Created:**
- `services` - Stores service registration data
- `subscriptions` - Stores pub/sub relationships

**Indexes:**
- `services.service_name` - For fast service group queries
- `services.status` - For health status filtering
- `subscriptions.service_group` - For subscriber lookups
- `subscriptions.subscriber_key + service_group` - Unique constraint

## Querying Service Pods

The manager provides convenient methods to query pods by service group:

```go
// Get all pods for a specific service
pods := mgr.GetServicePods("user-service")
for _, pod := range pods {
    fmt.Printf("Pod: %s, IP: %s, Status: %s\n",
        pod.PodName, pod.Providers[0].IP, pod.Status)
}

// Get all services and their pods
allServicePods := mgr.GetAllServicePods()
for serviceName, pods := range allServicePods {
    fmt.Printf("Service: %s has %d pods\n", serviceName, len(pods))
}
```

## Storage Performance Considerations

### In-Memory
- **Pros:** Fastest performance, no network latency
- **Cons:** Data lost on restart, single-node only
- **Use Case:** Development, testing, ephemeral environments

### MySQL/PostgreSQL
- **Pros:** ACID transactions, strong consistency, mature ecosystem
- **Cons:** Requires schema management, slightly slower than NoSQL
- **Use Case:** Production systems requiring strong consistency

### MongoDB
- **Pros:** Flexible schema, horizontal scaling, fast writes
- **Cons:** Eventual consistency in clustered setups
- **Use Case:** High-throughput systems, multi-datacenter deployments

## Connection Pool Settings

All database backends support connection pooling:

```go
config := mysql.Config{
    MaxOpenConns:    25,    // Maximum number of open connections
    MaxIdleConns:    5,     // Maximum number of idle connections
    ConnMaxLifetime: 5 * time.Minute, // Connection lifetime
}
```

Recommended settings:
- **Development:** MaxOpenConns=10, MaxIdleConns=2
- **Production (small):** MaxOpenConns=25, MaxIdleConns=5
- **Production (large):** MaxOpenConns=100, MaxIdleConns=10

## Error Handling

Storage operations may fail. The governance library handles errors gracefully:

- Failed writes log errors but don't crash the manager
- Failed reads return empty results
- Connection errors are logged during manager stop

For production systems, monitor storage health using the `Ping()` method:

```go
ctx := context.Background()
if err := store.Ping(ctx); err != nil {
    log.Printf("Storage unhealthy: %v", err)
}
```

## Examples

See the `examples/` directory for complete working examples:

- `examples/mysql_example/` - MySQL storage example
- `examples/postgresql_example/` - PostgreSQL storage example
- `examples/mongodb_example/` - MongoDB storage example
- `examples/query_pods_example/` - Querying pods by service group

## Custom Storage Implementation

You can implement your own storage backend by implementing the `RegistryStore` interface:

```go
type MyCustomStore struct {
    // Your storage implementation
}

func (s *MyCustomStore) SaveService(ctx context.Context, service *models.ServiceInfo) error {
    // Implementation
}

// Implement all other interface methods...

// Use with manager
store := &MyCustomStore{}
mgr := manager.NewManagerWithStorage(config, store)
```
