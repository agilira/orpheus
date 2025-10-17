# Orpheus: High-Performance CLI framework for Go

![Orpheus Banner](assets/banner.png)

[![CI/CD Pipeline](https://github.com/agilira/orpheus/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![CodeQL](https://github.com/agilira/orpheus/actions/workflows/codeql.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/codeql.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/orpheus?v=2)](https://goreportcard.com/report/github.com/agilira/orpheus)
[![Coverage](https://img.shields.io/badge/coverage-87.2%25-brightgreen.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11276/badge)](https://www.bestpractices.dev/projects/11276)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

Orpheus is a high-performance CLI framework designed to be super simple and **~30× faster** than popular alternatives with zero external dependencies. Built on [FlashFlags](https://github.com/agilira/flash-flags) & [go-errors](https://github.com/agilira/go-errors), Orpheus provides a simple interface to create modern, secure, fast CLI apps similar to git.

## Live Demo

<div align="center">

See Orpheus in action - building a Git-like CLI with subcommands in minutes:

<picture>
  <source media="(max-width: 768px)" srcset="https://asciinema.org/a/JiLb3gf6KzYU3VldOYjD4q1Zv.svg" width="100%">
  <source media="(max-width: 1024px)" srcset="https://asciinema.org/a/JiLb3gf6KzYU3VldOYjD4q1Zv.svg" width="90%">
  <img src="https://asciinema.org/a/JiLb3gf6KzYU3VldOYjD4q1Zv.svg" alt="Orpheus CLI Demo" style="max-width: 100%; height: auto;" width="800">
</picture>

*[Click to view interactive demo](https://asciinema.org/a/JiLb3gf6KzYU3VldOYjD4q1Zv)*

</div>

**[Features](#features) • [Performance](#performance) • [Security](#security) • [Quick Start](#quick-start) • [Storage System](#storage-system) • [Observability](#production-observability) • [Examples](#examples) • [API Reference](#api-reference) • [Philosophy](#the-philosophy-behind-orpheus)**

## Features

- **Zero External Dependencies**: No third-party dependencies for maximum portability
- **Native Subcommands**: Git-style nested commands with automatic help generation
- **Pluggable Storage System**: Dynamic .so plugin loading for persistent storage (SQLite, Redis, File, custom providers)
- **Clean API**: Fluent interface for rapid development
- **Auto-completion**: Built-in bash/zsh/fish completion generation
- **Type-safe Errors**: Structured error handling with exit codes
- **Hot-swappable Commands**: Dynamic command registration and modification
- **Production Observability**: Zero-overhead logging, audit trails, tracing, and metrics interfaces
- **Secure by Design**: [Red-team tested](./pkg/orpheus/security_test.go) and [fuzz tested](./pkg/orpheus/orpheus_fuzz_test.go)
- **Security Validation**: Including input sanitization, path traversal protection, and various security controls

## Compatibility and Support

Orpheus is designed for Go 1.23+ environments and follows Long-Term Support guidelines to ensure consistent performance across production deployments.

## Performance

Benchmark results comparing CLI framework performance:

```
AMD Ryzen 5 7520U with Radeon Graphics
BenchmarkOrpheus-8       1908495           634.5 ns/op          96 B/op       3 allocs/op
BenchmarkCobra-8              66        18439562 ns/op        3145 B/op      33 allocs/op
BenchmarkUrfaveCli-8       40767           30097 ns/op        8549 B/op     318 allocs/op
BenchmarkKingpin-8        293697           4294 ns/op         1988 B/op      40 allocs/op
BenchmarkStdFlag-8       1027216           1039 ns/op          945 B/op      13 allocs/op
```

**Scenario**: Command parsing with 3 flags (string, bool, string) and handler execution.

**Reproduce benchmarks**:
```bash
cd benchmarks/
go test -bench=. -benchmem
```

**[Complete Performance Benchmarks →](./benchmarks/benchmark_test.go)**

## Security

Orpheus implements defense-in-depth security with comprehensive validation against CLI attack vectors.

**Protected Vectors:**
- Path traversal (case-insensitive, URL encoding, Windows device names)
- Command/SQL/Script injection prevention
- Control character and null byte filtering
- Cross-platform consistency (Windows, Linux, macOS)

**Run Security Tests:**
```bash
make security      # Run security test suite
make fuzz          # Quick fuzz testing (30s)
make fuzz-long     # Extended fuzzing (5min)
```

## Quick Start

### Installation

```bash
go get github.com/agilira/orpheus@v1.0.10  # Latest stable release
# or simply
go get github.com/agilira/orpheus          # Always latest
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/agilira/orpheus"
)

func main() {
    app := orpheus.New("myapp").
        SetDescription("My awesome CLI application").
        SetVersion("1.0.0")

    // Add commands with fluent interface
    app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
        fmt.Println("Service starting...")
        return nil
    })

    app.Command("stop", "Stop the service", func(ctx *orpheus.Context) error {
        fmt.Println("Service stopping...")
        return nil
    })

    // Run the application
    if err := app.Run(os.Args[1:]); err != nil {
        log.Fatal(err)
    }
}
```

### Auto-completion

Generate shell completion scripts for your CLI:

```bash
# Bash completion (add to ~/.bashrc)
./myapp completion bash > /etc/bash_completion.d/myapp

# Zsh completion (add to ~/.zshrc)
./myapp completion zsh > /usr/local/share/zsh/site-functions/_myapp

# Fish completion
./myapp completion fish > ~/.config/fish/completions/myapp.fish
```

### Subcommands

```go
// Create a command with subcommands
remoteCmd := orpheus.NewCommand("remote", "Manage remote repositories")

// Add subcommands using fluent API (v1.0.7+ - now works correctly)
remoteCmd.Subcommand("add", "Add a remote", func(ctx *orpheus.Context) error {
    name, url := ctx.GetArg(0), ctx.GetArg(1)
    fmt.Printf("Added remote: %s -> %s\n", name, url)
    return nil
}).AddFlag("--force", "Force add remote")

remoteCmd.Subcommand("list", "List remotes", func(ctx *orpheus.Context) error {
    fmt.Println("origin\thttps://github.com/user/repo.git")
    return nil
}).AddFlag("--verbose", "Show detailed information")

app.AddCommand(remoteCmd)

// Usage: ./myapp remote add --force origin https://github.com/user/repo.git
//        ./myapp remote list --verbose
```

## Storage System

Orpheus provides a pluggable storage system that allows CLI applications to persist state using various backends through a unified interface:

```go
import "github.com/agilira/orpheus/pkg/orpheus"

func main() {
    app := orpheus.New("myapp").
        SetDescription("CLI app with persistent storage").
        SetVersion("1.0.0")
    
    // Configure storage (supports SQLite, Redis, File, and custom providers)
    config := &orpheus.StorageConfig{
        Provider: "sqlite",
        Config: map[string]interface{}{
            "path": "./myapp.db",
        },
        EnableMetrics: true,
    }
    app.ConfigureStorage(config)
    
    // Commands can now use persistent storage
    app.Command("set", "Store a key-value pair", setCommand)
    app.Command("get", "Retrieve a value", getCommand)
    
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

**Key Features:**
- **Plugin Architecture**: Dynamic .so loading for storage providers
- **Zero Dependencies**: No external storage libraries required  
- **Security Hardened**: Input validation and plugin security checks
- **Production Ready**: Metrics, tracing, and audit logging integration
- **Provider Agnostic**: Unified interface for SQLite, Redis, File, and custom backends

**[Complete Storage Documentation →](./docs/STORAGE.md)**

### Observability

Zero-overhead observability interfaces for production CLI applications:

```go
import "context"

// Configure observability (all interfaces are optional)
app := orpheus.New("myapp").
    SetLogger(myLogger).           // Structured logging
    SetAuditLogger(myAuditLogger). // Compliance and security
    SetTracer(myTracer).           // Distributed tracing
    SetMetricsCollector(myMetrics) // Performance metrics

app.Command("deploy", "Deploy application", func(ctx *orpheus.Context) error {
    // Structured logging
    if logger := ctx.Logger(); logger != nil {
        logger.Info(context.Background(), "Deployment started",
            orpheus.StringField("environment", "production"),
            orpheus.StringField("version", "v1.2.3"),
        )
    }

    // Audit trail
    if audit := ctx.AuditLogger(); audit != nil {
        audit.LogCommand(context.Background(), "deploy", ctx.Args(), "demo-user")
        audit.LogAccess(context.Background(), "production", "deploy", true)
    }

    // Distributed tracing
    if tracer := ctx.Tracer(); tracer != nil {
        spanCtx, span := tracer.StartSpan(context.Background(), "deploy_operation")
        defer span.End()
        // ... use spanCtx for downstream operations
    }

    // Performance metrics
    if metrics := ctx.MetricsCollector(); metrics != nil {
        counter := metrics.Counter("deployments_total", "Total deployments", "env")
        counter.Inc(context.Background(), "production")
    }

    fmt.Println("Deployment completed")
    return nil
})
```

**Performance**: Zero overhead when not configured (0.24 ns/op), minimal overhead when enabled (~24 ns/op)

**[Complete Observability Guide →](./docs/OBSERVABILITY.md)**

**[Complete Examples →](./docs/EXAMPLES.md)**

## The Philosophy Behind Orpheus

Orpheus's lyre was no ordinary instrument. It could make rivers pause mid-flow, convince stones to dance, and move even Hades and Persephone to tears. When the great musician played, the impossible became inevitable—not through force, but through the pure beauty of perfect harmony.

Yet Orpheus understood something profound: true power lay not in complexity, but in elegant simplicity. Each note served a purpose, each silence held meaning. His melodies moved the world through perfect clarity of intent, never through force or intricacy.

Like the mythological master whose music could open the gates of Hades itself, Orpheus, transforms the cacophony of human commands into pure, executable harmony, making the complex dance between intention and fulfillment beautifully simple.

### Core Components

- **App**: Main application container with command routing
- **Command**: Individual command with handler and flags
- **Context**: Execution context with arguments and flags
- **Errors**: Type-safe error system with exit codes
- **Completion**: Auto-completion system for multiple shells
- **Observability**: Optional interfaces for logging, audit trails, tracing, and metrics

## Testing

```bash
# Run all tests
go test ./pkg/orpheus -v

# Run with coverage
go test ./pkg/orpheus -v -cover
```

## Examples

**[Complete Examples →](./docs/EXAMPLES.md)**

**Practical implementations:**
- **[GitLike Demo](./examples/gitlike/)** - Git-style CLI with subcommands and JSON persistence
- **[File Manager](./examples/filemanager/)** - Advanced file operations with completion
- **[Enhanced Errors](./examples/enhanced-errors/)** - Advanced errors handling
- **[Observability](./examples/observability/)** - Production-ready logging, audit trails, and metrics
- **[Storage System](./examples/storage/)** - Persistent storage
- **[Basic Example](./examples/basic/)** - Simple usage patterns and command structures

## API Reference

**[Complete API Reference →](./docs/API.md)**

## License

Orpheus is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

Orpheus • an AGILira library
