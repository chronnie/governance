# Governance Logger

The governance library uses [uber-go/zap](https://github.com/uber-go/zap) for structured, high-performance logging.

## Features

- **Disabled by default** - Zero overhead when not needed
- **Environment-based configuration** - Easy to enable for debugging
- **Structured logging** - JSON or console output
- **Configurable log levels** - debug, info, warn, error

## Configuration

Logging is controlled by environment variables:

### `GOVERNANCE_LOG_ENABLED`

Enable or disable logging:
- `true` - Logging enabled
- `false` or unset - Logging disabled (default, no-op logger)

```bash
export GOVERNANCE_LOG_ENABLED=true
```

### `GOVERNANCE_LOG_LEVEL`

Set the minimum log level:
- `debug` - Most verbose, shows all logs
- `info` - Normal operational messages (default)
- `warn` - Warning messages
- `error` - Error messages only

```bash
export GOVERNANCE_LOG_LEVEL=debug
```

### `GOVERNANCE_LOG_FORMAT`

Set the log output format:
- `console` - Human-readable format (default)
- `json` - JSON format for log aggregation

```bash
export GOVERNANCE_LOG_FORMAT=json
```

## Usage Examples

### Basic Usage

```go
import "github.com/chronnie/governance/pkg/logger"

// Simple logging
logger.Info("Service started")
logger.Error("Failed to connect")

// Formatted logging
logger.Infof("Server listening on port %d", 8080)
logger.Errorf("Connection failed: %v", err)
```

### Structured Logging

```go
import (
    "github.com/chronnie/governance/pkg/logger"
    "go.uber.org/zap"
)

// Add structured fields
logger.Info("Service registered",
    zap.String("service_name", "user-service"),
    zap.String("pod_name", "user-pod-1"),
    zap.Int("port", 8080),
)

logger.Error("Health check failed",
    zap.String("service", "order-service"),
    zap.Error(err),
    zap.Duration("timeout", 5*time.Second),
)
```

### With Context Fields

```go
// Create logger with persistent fields
serviceLogger := logger.WithFields(
    zap.String("service", "user-service"),
    zap.String("pod", "user-pod-1"),
)

serviceLogger.Info("Pod started")
serviceLogger.Warn("High memory usage")
```

## Enable Logging for Development

```bash
# Linux/Mac
export GOVERNANCE_LOG_ENABLED=true
export GOVERNANCE_LOG_LEVEL=debug
export GOVERNANCE_LOG_FORMAT=console

# Windows PowerShell
$env:GOVERNANCE_LOG_ENABLED="true"
$env:GOVERNANCE_LOG_LEVEL="debug"
$env:GOVERNANCE_LOG_FORMAT="console"

# Windows CMD
set GOVERNANCE_LOG_ENABLED=true
set GOVERNANCE_LOG_LEVEL=debug
set GOVERNANCE_LOG_FORMAT=console
```

## Enable Logging for Production

For production environments with log aggregation:

```bash
export GOVERNANCE_LOG_ENABLED=true
export GOVERNANCE_LOG_LEVEL=info
export GOVERNANCE_LOG_FORMAT=json
```

## Docker/Kubernetes

### Docker

```dockerfile
ENV GOVERNANCE_LOG_ENABLED=true
ENV GOVERNANCE_LOG_LEVEL=info
ENV GOVERNANCE_LOG_FORMAT=json
```

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: governance-manager
spec:
  containers:
  - name: manager
    image: governance-manager:latest
    env:
    - name: GOVERNANCE_LOG_ENABLED
      value: "true"
    - name: GOVERNANCE_LOG_LEVEL
      value: "info"
    - name: GOVERNANCE_LOG_FORMAT
      value: "json"
```

## Log Locations

The library logs in the following locations:

### Manager
- Service startup/shutdown
- HTTP server status
- Event queue errors
- Storage connection errors

### Worker
- Database synchronization (debug level)
- Event processing (if errors occur)

### Registry
- Service registration/unregistration
- Health status updates

## Performance

When `GOVERNANCE_LOG_ENABLED=false` (default):
- Uses `zap.NewNop()` - a no-op logger
- **Zero allocation overhead**
- **Zero CPU overhead**
- No performance impact on production systems

When enabled:
- Zap is one of the fastest Go logging libraries
- Minimal allocation and overhead
- Suitable for production use

## Examples

See the examples directory for complete usage:
- `examples/manager_example/` - Basic logging setup
- `examples/mysql_example/` - Logging with database
- `examples/postgresql_example/` - Logging with PostgreSQL
- `examples/mongodb_example/` - Logging with MongoDB

## Best Practices

1. **Default to disabled** - Don't enable logging unless debugging
2. **Use structured fields** - Prefer `zap.String()` over string formatting
3. **Appropriate log levels** - Use debug for verbose, error for failures
4. **Flush on shutdown** - Call `logger.Sync()` before exit (done automatically in manager)
