# Orpheus Storage System

The Orpheus Storage System provides pluggable persistent storage capabilities for CLI applications, enabling sophisticated state management with zero-overhead when unused.

## Overview

The storage system is built on three core principles:

1. **Zero External Dependencies** - Pure Go implementation using only standard library
2. **Plugin Architecture** - Dynamic loading of storage providers via .so files  
3. **Production Ready** - Thread-safe, observable, and security-hardened

## Quick Start

### Basic Configuration

```go
package main

import (
    "github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
    app := orpheus.New("myapp").
        SetDescription("My CLI application with storage").
        SetVersion("1.0.0")
    
    // Configure storage
    config := &orpheus.StorageConfig{
        Provider: "sqlite", 
        Config: map[string]interface{}{
            "path": "./app.db",
        },
    }
    app.ConfigureStorage(config)
    
    // Add commands that use storage
    app.Command("set", "Set a key-value pair", setCommand).
    app.Command("get", "Get a value by key", getCommand)
    
    app.Run()
}

func setCommand(ctx *orpheus.Context) error {
    storage := ctx.Storage()
    if storage == nil {
        return orpheus.ErrStorageNotConfigured
    }
    
    key := ctx.GetArg(0)
    value := ctx.GetArg(1)
    
    return storage.Set(ctx, key, []byte(value))
}

func getCommand(ctx *orpheus.Context) error {
    storage, err := ctx.RequireStorage()
    if err != nil {
        return err
    }
    
    key := ctx.GetArg(0)
    value, err := storage.Get(ctx, key)
    if err != nil {
        return err
    }
    
    fmt.Printf("Value: %s\n", string(value))
    return nil
}
```

## Architecture

### Storage Interface

All storage providers implement the `Storage` interface:

```go
type Storage interface {
    // Basic Operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
    
    // Health & Monitoring
    Health(ctx context.Context) error
    Stats(ctx context.Context) (*StorageStats, error)
    Close() error
}
```

### Plugin System

Storage providers are loaded as plugins implementing `StoragePlugin`:

```go
type StoragePlugin interface {
    Name() string
    Version() string
    Description() string
    New(config map[string]interface{}) (Storage, error)
    Validate(config map[string]interface{}) error
    DefaultConfig() map[string]interface{}
}
```

## Configuration Options

### StorageConfig Structure

```go
type StorageConfig struct {
    // Provider specifies which storage provider to use
    Provider string `json:"provider" yaml:"provider"`
    
    // PluginPath is the path to the provider plugin (.so file)
    // If empty, auto-discovery is used
    PluginPath string `json:"plugin_path,omitempty" yaml:"plugin_path,omitempty"`
    
    // Config contains provider-specific configuration
    Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
    
    // Namespace provides key isolation for multi-app scenarios
    Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
    
    // Observability Controls
    EnableMetrics bool `json:"enable_metrics,omitempty" yaml:"enable_metrics,omitempty"`
    EnableTracing bool `json:"enable_tracing,omitempty" yaml:"enable_tracing,omitempty"`
    EnableAudit   bool `json:"enable_audit,omitempty" yaml:"enable_audit,omitempty"`
}
```

### Configuration Examples

#### SQLite Storage
```go
config := &orpheus.StorageConfig{
    Provider: "sqlite",
    Config: map[string]interface{}{
        "path":            "./myapp.db",
        "journal_mode":    "WAL",
        "synchronous":     "NORMAL",
        "cache_size":      "10000",
        "timeout":         "30s",
    },
    EnableMetrics: true,
    EnableAudit:   true,
}
```

#### File-based Storage
```go
config := &orpheus.StorageConfig{
    Provider: "file",
    Config: map[string]interface{}{
        "directory":       "./data",
        "file_extension":  ".json",
        "create_dirs":     true,
        "sync_writes":     true,
    },
    Namespace: "myapp",
}
```

#### Redis Storage
```go
config := &orpheus.StorageConfig{
    Provider: "redis",
    PluginPath: "/opt/plugins/redis.so",
    Config: map[string]interface{}{
        "host":        "localhost",
        "port":        6379,
        "password":    "",
        "database":    0,
        "pool_size":   10,
        "timeout":     "5s",
    },
}
```

#### Configuration from File
```go
// Load from JSON
var config orpheus.StorageConfig
data, _ := os.ReadFile("storage.json")
json.Unmarshal(data, &config)
app.ConfigureStorage(&config)

// Load from YAML
var config orpheus.StorageConfig
data, _ := os.ReadFile("storage.yaml") 
yaml.Unmarshal(data, &config)
app.ConfigureStorage(&config)
```

## Plugin Development

### Creating a Storage Plugin

1. **Implement StoragePlugin Interface**

```go
// plugin/sqlite/plugin.go
package main

import "C"

type SQLitePlugin struct{}

func (p *SQLitePlugin) Name() string {
    return "sqlite"
}

func (p *SQLitePlugin) Version() string {
    return "1.0.0"
}

func (p *SQLitePlugin) Description() string {
    return "SQLite storage provider for Orpheus"
}

func (p *SQLitePlugin) New(config map[string]interface{}) (Storage, error) {
    path, ok := config["path"].(string)
    if !ok {
        return nil, fmt.Errorf("path is required")
    }
    
    return NewSQLiteStorage(path, config)
}

func (p *SQLitePlugin) Validate(config map[string]interface{}) error {
    if _, ok := config["path"].(string); !ok {
        return fmt.Errorf("path must be a string")
    }
    return nil
}

func (p *SQLitePlugin) DefaultConfig() map[string]interface{} {
    return map[string]interface{}{
        "path":         "./app.db",
        "journal_mode": "WAL",
        "synchronous":  "NORMAL",
    }
}

// Export symbol for plugin system
//export NewStoragePlugin
func NewStoragePlugin() StoragePlugin {
    return &SQLitePlugin{}
}

func main() {} // Required for plugin
```

2. **Implement Storage Interface**

```go
// plugin/sqlite/storage.go
type SQLiteStorage struct {
    db    *sql.DB
    mutex sync.RWMutex
    stats *StorageStats
}

func NewSQLiteStorage(path string, config map[string]interface{}) (*SQLiteStorage, error) {
    // Implementation details...
}

func (s *SQLiteStorage) Get(ctx context.Context, key string) ([]byte, error) {
    // Implementation with proper context handling, tracing, metrics
}

// ... implement all Storage interface methods
```

3. **Build Plugin**

```bash
# Build as shared library
go build -buildmode=plugin -o sqlite.so ./plugin/sqlite/

# Install to standard location
sudo mkdir -p /usr/local/lib/orpheus/plugins
sudo cp sqlite.so /usr/local/lib/orpheus/plugins/
```

### Plugin Discovery

Plugins are discovered in these locations (in order):

1. **Explicit Path** - `StorageConfig.PluginPath`
2. **Standard Locations**:
   - `/usr/local/lib/orpheus/plugins/`
   - `/opt/orpheus/plugins/`  
   - `./plugins/`
   - `~/.orpheus/plugins/`

## Error Handling

### Structured Error System

The storage system uses AGILira's structured error handling:

```go
// Check error types
if orpheus.IsStorageNotFound(err) {
    // Handle key not found
    fmt.Println("Key not found")
    return nil
}

if orpheus.IsStorageUnavailable(err) {
    // Handle temporary failures (retryable)
    fmt.Println("Storage temporarily unavailable, retrying...")
    time.Sleep(time.Second)
    // retry operation
}

if orpheus.IsStorageValidationError(err) {
    // Handle configuration errors
    fmt.Printf("Configuration error: %v\n", err)
    return err
}
```

### Error Codes

| Code | Description | Severity | Retryable |
|------|-------------|----------|-----------|
| `ORF2000` | Validation Error | Error | No |
| `ORF2001` | Execution Error | Warning/Error | Maybe |
| `ORF2002` | Key Not Found | Info | No |
| `ORF2003` | Storage Unavailable | Critical | Yes |
| `ORF2004` | Plugin Error | Critical | No |

## Observability

### Metrics Integration

```go
// Enable metrics collection
config := &orpheus.StorageConfig{
    Provider:      "sqlite",
    EnableMetrics: true,
    Config:        map[string]interface{}{"path": "./app.db"},
}
app.ConfigureStorage(config)

// Access metrics
storage := ctx.Storage()
stats, err := storage.Stats(ctx)
if err == nil {
    fmt.Printf("Total operations: %d\n", stats.GetOperations)
    fmt.Printf("Average latency: %v\n", stats.AvgGetLatency)
    fmt.Printf("Error rate: %.2f%%\n", 
        float64(stats.GetErrors)/float64(stats.GetOperations)*100)
}
```

### Tracing Integration

```go
// Enable distributed tracing
app.SetTracer(myTracer).ConfigureStorage(&orpheus.StorageConfig{
    Provider:      "sqlite", 
    EnableTracing: true,
    Config:        map[string]interface{}{"path": "./app.db"},
})

// Storage operations automatically create spans
func myCommand(ctx *orpheus.Context) error {
    // This creates a span: "storage.Get"
    value, err := ctx.Storage().Get(ctx, "user:123")
    return err
}
```

### Audit Logging

```go
// Enable security auditing
app.SetAuditLogger(myAuditLogger).ConfigureStorage(&orpheus.StorageConfig{
    Provider:    "sqlite",
    EnableAudit: true, 
    Config:      map[string]interface{}{"path": "./app.db"},
})

// All storage operations are logged with:
// - Operation type (GET/SET/DELETE/LIST)
// - Key accessed (with optional redaction)
// - User context
// - Success/failure status
// - Performance metrics
```

## Security

### Security Configuration

```go
// Configure plugin security
securityConfig := &orpheus.PluginSecurityConfig{
    AllowUnsignedPlugins: false,           // Require signed plugins
    ValidateChecksums:    true,            // Verify plugin integrity  
    MaxPluginSize:        50 << 20,        // 50MB max plugin size
    AllowedPaths: []string{                // Restrict plugin locations
        "/usr/local/lib/orpheus/plugins",
        "/opt/orpheus/plugins",
    },
    RequiredSymbols: []string{             // Validate plugin exports
        "NewStoragePlugin",
    },
}

// Create plugin manager with security config
pluginManager := orpheus.NewPluginManager(logger, securityConfig)
```

### Key Validation

```go
// Keys are automatically validated for security
err := storage.Set(ctx, "../../../etc/passwd", data) 
// Returns: validation error (path traversal detected)

err := storage.Set(ctx, "user\x00admin", data)
// Returns: validation error (null byte injection detected) 

err := storage.Set(ctx, strings.Repeat("x", 10000), data)
// Returns: validation error (key too long)
```

## Performance Optimization

### Connection Pooling

```go
// Configure connection pooling for database providers
config := &orpheus.StorageConfig{
    Provider: "postgres",
    Config: map[string]interface{}{
        "host":          "localhost",
        "port":          5432,
        "database":      "myapp",
        "max_conns":     25,           // Connection pool size
        "max_idle":      5,            // Idle connections
        "conn_lifetime": "1h",         // Connection lifetime
    },
}
```

### Caching

```go
// Enable provider-level caching 
config := &orpheus.StorageConfig{
    Provider: "redis",
    Config: map[string]interface{}{
        "host":        "localhost",
        "port":        6379,
        "cache_size":  10000,          // In-memory cache entries
        "cache_ttl":   "5m",           // Cache TTL
        "compress":    true,           // Compress large values
    },
}
```

### Batch Operations

```go
// Use batch operations for better performance
func bulkInsert(ctx *orpheus.Context, data map[string][]byte) error {
    storage := ctx.Storage()
    
    // Some providers support batch operations
    if batcher, ok := storage.(BatchStorage); ok {
        return batcher.SetBatch(ctx, data)
    }
    
    // Fallback to individual operations
    for key, value := range data {
        if err := storage.Set(ctx, key, value); err != nil {
            return err
        }
    }
    return nil
}
```

## Testing

### Unit Testing with Mock Storage

```go
func TestMyCommand(t *testing.T) {
    // Create mock storage
    mockStorage := orpheus.NewMockStorage()
    mockStorage.Set(context.Background(), "test_key", []byte("test_value"))
    
    // Create test app
    app := orpheus.New("testapp").SetStorage(mockStorage)
    
    // Create test context
    ctx := &orpheus.Context{
        App:     app,
        Args:    []string{"test_key"},
        storage: mockStorage,
    }
    
    // Test command
    err := myCommand(ctx)
    assert.NoError(t, err)
}
```

### Integration Testing

```go
func TestStorageIntegration(t *testing.T) {
    // Use temporary directory
    tempDir := t.TempDir()
    
    config := &orpheus.StorageConfig{
        Provider: "sqlite",
        Config: map[string]interface{}{
            "path": filepath.Join(tempDir, "test.db"),
        },
    }
    
    app := orpheus.New("testapp").ConfigureStorage(config) 
    
    // Test storage operations
    storage := app.Storage()
    ctx := context.Background()
    
    err := storage.Set(ctx, "key1", []byte("value1"))
    assert.NoError(t, err)
    
    value, err := storage.Get(ctx, "key1")
    assert.NoError(t, err)
    assert.Equal(t, []byte("value1"), value)
}
```

## Migration & Compatibility

### Schema Migrations

```go
// Handle schema migrations in plugin initialization
func (s *SQLiteStorage) migrate() error {
    version, err := s.getSchemaVersion()
    if err != nil {
        return err
    }
    
    migrations := []Migration{
        {Version: 1, SQL: "CREATE TABLE kv_store (key TEXT PRIMARY KEY, value BLOB)"},
        {Version: 2, SQL: "CREATE INDEX idx_key_prefix ON kv_store(key)"},
        {Version: 3, SQL: "ALTER TABLE kv_store ADD COLUMN created_at INTEGER"},
    }
    
    for _, migration := range migrations {
        if version < migration.Version {
            if err := s.runMigration(migration); err != nil {
                return err
            }
        }
    }
    return nil
}
```

### Backward Compatibility

Storage plugins follow semantic versioning. Breaking changes require major version bumps:

```go
// Version compatibility checking
func (p *SQLitePlugin) Version() string {
    return "2.1.0" // Major.Minor.Patch
}

// Plugin manager validates compatibility
pluginManager.LoadPlugin(ctx, "/path/to/plugin.so")
// Automatically checks: orpheus_version >= required_version
```

## Best Practices

### Application Design

1. **Graceful Degradation**
```go
func myCommand(ctx *orpheus.Context) error {
    storage := ctx.Storage()
    if storage == nil {
        // App works without storage (reduced functionality)
        fmt.Println("Storage not configured, using defaults")
        return nil
    }
    
    // Use storage when available
    // ...
}
```

2. **Error Recovery**
```go
func robustGet(ctx *orpheus.Context, key string) ([]byte, error) {
    storage, err := ctx.RequireStorage()
    if err != nil {
        return nil, err
    }
    
    value, err := storage.Get(ctx, key)
    if orpheus.IsStorageUnavailable(err) {
        // Retry with exponential backoff
        return retryWithBackoff(ctx, storage, key)
    }
    return value, err
}
```

3. **Resource Management**
```go
func main() {
    app := orpheus.New("myapp").ConfigureStorage(config)
    defer func() {
        if storage := app.Storage(); storage != nil {
            storage.Close()
        }
    }()
    
    app.Run()
}
```

### Performance Guidelines

1. **Batch Operations**: Use bulk operations when available
2. **Connection Pooling**: Configure appropriate pool sizes  
3. **Caching**: Enable caching for read-heavy workloads
4. **Key Design**: Use hierarchical keys for efficient prefix queries
5. **Monitoring**: Enable metrics and tracing in production

### Security Guidelines

1. **Input Validation**: All keys and values are validated automatically
2. **Plugin Security**: Use signed plugins in production
3. **Access Control**: Implement application-level access controls
4. **Audit Logging**: Enable audit trails for sensitive operations
5. **Network Security**: Use TLS for remote storage connections

## Troubleshooting

### Common Issues

#### Plugin Loading Failures
```bash
# Check plugin file
file /path/to/plugin.so
# Should show: ELF 64-bit LSB shared object

# Check plugin symbols  
nm -D /path/to/plugin.so | grep NewStoragePlugin
# Should show the exported symbol

# Check plugin dependencies
ldd /path/to/plugin.so
# Verify all dependencies are available
```

#### Storage Connection Issues
```go
// Enable detailed logging
logger := orpheus.NewLogger().SetLevel("DEBUG")
app.SetLogger(logger).ConfigureStorage(config)

// Check storage health
storage := app.Storage()
if storage != nil {
    err := storage.Health(context.Background())
    if err != nil {
        log.Printf("Storage health check failed: %v", err)
    }
}
```

#### Performance Issues  
```go
// Analyze storage statistics
stats, _ := storage.Stats(ctx)
fmt.Printf("Average latencies:\n")
fmt.Printf("  Get: %v\n", stats.AvgGetLatency)
fmt.Printf("  Set: %v\n", stats.AvgSetLatency)
fmt.Printf("Error rates:\n")
fmt.Printf("  Get: %.2f%%\n", float64(stats.GetErrors)/float64(stats.GetOperations)*100)
```

## Examples Repository

Complete examples are available at: https://github.com/agilira/orpheus/examples/storage

- **Basic Usage**: Simple key-value operations
- **Configuration**: Multiple configuration approaches
- **Plugin Development**: Complete plugin implementation
- **Testing**: Unit and integration test examples
- **Production**: Production-ready configuration examples

---

Orpheus • an AGILira library