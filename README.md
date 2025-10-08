# Orpheus: High-Performance zero external dependencies CLI framework for Go
### an AGILira library

[![CI/CD Pipeline](https://github.com/agilira/orpheus/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![CodeQL](https://github.com/agilira/orpheus/actions/workflows/codeql.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/codeql.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/orpheus?v=2)](https://goreportcard.com/report/github.com/agilira/orpheus)
[![Coverage](https://codecov.io/gh/agilira/orpheus/branch/main/graph/badge.svg)](https://codecov.io/gh/agilira/orpheus)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11276/badge)](https://www.bestpractices.dev/projects/11276)

Orpheus is a high-performance CLI framework designed to be super simple and **7x-53x faster** than popular alternatives with zero external dependencies. Built on [FlashFlags](https://github.com/agilira/flash-flags), Orpheus provides a simple interface to create modern, fast CLI apps similar to git.

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

**[Features](#features) • [Performance](#performance) • [Security](#security-assurance) • [Quick Start](#quick-start) • [Observability](#production-observability) • [Examples](#examples) • [API Reference](#api-reference) • [Philosophy](#the-philosophy-behind-orpheus)**

## Features

- **Zero External Dependencies**: No third-party dependencies for maximum portability
- **Native Subcommands**: Git-style nested commands with automatic help generation
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
BenchmarkOrpheus-8       2283835           512.8 ns/op          96 B/op          3 allocs/op
BenchmarkCobra-8          279333           3727 ns/op           1752 B/op        29 allocs/op
BenchmarkUrfaveCli-8       41664           28026 ns/op          9334 B/op        366 allocs/op
BenchmarkKingpin-8        294334           3419 ns/op           1988 B/op        40 allocs/op
BenchmarkStdFlag-8       1476625           809.0 ns/op          945 B/op         13 allocs/op
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

**Security Validation:**
- [142+ Red Team security test cases](./pkg/orpheus/security_test.go) (OWASP Top 10 coverage)
- [3.5+ million fuzz test executions](./pkg/orpheus/orpheus_fuzz_test.go) with zero crashes
- CodeQL static analysis with security-focused queries
- govulncheck for known vulnerability scanning
- Multi-layer protection: path traversal, injection attacks, Unicode bypasses

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
go get github.com/agilira/orpheus
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/agilira/orpheus/pkg/orpheus"
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
- **[Basic Demo](./cmd/demo/)** - Simple usage patterns and command structures

## API Reference

**[Complete API Reference →](./docs/API.md)**

## License

Orpheus is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

Orpheus • an AGILira library
