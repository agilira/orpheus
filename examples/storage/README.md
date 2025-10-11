# Orpheus Plugin Storage System Demo

This example demonstrates Orpheus's **complete plugin architecture** for storage systems:

1. **🔌 Dynamic Plugin Loading** - Real .so plugin compilation and loading
2. **🔒 Plugin Security System** - Path validation, integrity checks, symbol verification
3. **⚡ High-Performance Architecture** - Zero-overhead plugin interface
4. **🏗️ Production Plugin Development** - Complete plugin development workflow

## Plugin System Demo

```bash
# 1. Build storage plugins as .so files
./build_plugins.sh

# 2. Build the demo application  
go build -o storage-demo .

# 3. Test plugin loading and operations
./storage-demo info          # Shows plugin loaded successfully
./storage-demo set key value # Each command loads plugin fresh
./storage-demo benchmark     # Performance with plugin overhead

# 4. Security validation
./storage-demo security-test --full
```

## What This Demonstrates

- **Real Plugin Compilation**: `memory.go` → `memory.so` shared library
- **Secure Plugin Discovery**: Path traversal protection, integrity validation
- **Dynamic Symbol Resolution**: Runtime loading of plugin functions  
- **Plugin Interface Compliance**: StoragePlugin interface implementation
- **Production Plugin Architecture**: Complete plugin development lifecycle

## Security Features

This example showcases Orpheus's comprehensive security model:

- **Plugin Path Validation** - Prevents path traversal attacks
- **Binary Integrity Checks** - SHA256 validation of plugin files
- **Symbol Resolution Security** - Validates required plugin symbols
- **Memory Safety** - Bounded operations preventing DoS attacks
- **Concurrent Access Protection** - Thread-safe plugin management
- **Error Information Sanitization** - No sensitive data leakage

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Application                         │
├─────────────────────────────────────────────────────────────┤
│                   Orpheus Framework                         │
│  ┌─────────────────┐  ┌─────────────────┐                 │
│  │  Plugin Manager │  │ Security Layer  │                 │
│  │  - Discovery    │  │ - Path Validation│                │
│  │  - Loading      │  │ - Symbol Check  │                 │
│  │  - Lifecycle    │  │ - Integrity     │                 │
│  └─────────────────┘  └─────────────────┘                 │
├─────────────────────────────────────────────────────────────┤
│              Dynamic Storage Plugins (.so)                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Memory    │  │    File     │  │   SQLite    │        │
│  │  Provider   │  │  Provider   │  │  Provider   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Performance Benchmarks

The storage system achieves sub-nanosecond performance for critical operations:

- **Storage Access**: ~0.26 ns/op (0 allocations)
- **Plugin Management**: ~53 ns/op (1 allocation)  
- **Security Validation**: ~2.2 μs/operation
- **Hash Calculation**: ~147 ns/operation

## Security Testing

Run the comprehensive security test suite:

```bash
# Red team security tests
go test -v -run "TestSecurity.*"

# Fuzzing with adversarial inputs
go test -fuzz=FuzzPluginPathSecurity -fuzztime=30s
go test -fuzz=FuzzPluginBinaryContent -fuzztime=30s
go test -fuzz=FuzzConcurrentPluginOperations -fuzztime=30s
```

## Production Deployment

For production use, configure strict security policies:

```go
config := &orpheus.PluginSecurityConfig{
    AllowUnsignedPlugins: false,    // Require signed plugins
    ValidateChecksums:    true,     // Enable integrity checks
    MaxPluginSize:        50 << 20, // 50MB max plugin size
    AllowedPaths: []string{
        "/opt/myapp/plugins",       // Restrict plugin locations
    },
    RequiredSymbols: []string{
        "NewStoragePlugin",         // Required plugin interface
    },
}
```

## Files

- `main.go` - Main application with CLI interface
- `providers/` - Example storage provider implementations
- `plugins/` - Pre-built plugin binaries for testing
- `security_test.go` - Comprehensive security validation
- `benchmark_test.go` - Performance validation tests

---

Orpheus • an AGILira library