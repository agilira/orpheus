# Orpheus: High-Performance zero-dependency CLI framework for Go
### an AGILira library

[![CI/CD Pipeline](https://github.com/agilira/orpheus/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/orpheus?v=2)](https://goreportcard.com/report/github.com/agilira/orpheus)
[![Coverage](https://codecov.io/gh/agilira/orpheus/branch/main/graph/badge.svg)](https://codecov.io/gh/agilira/orpheus)

Orpheus is a high-performance CLI framework designed to be super simple and **7x-53x faster** than popular alternatives with zero external dependencies. Orpheus provides a simple interface to create modern, fast CLI apps similar to git.

## Features

- **Zero Dependencies**: Zero external dependencies for maximum portability
- **Native Subcommands**: Git-style nested commands with automatic help generation
- **Clean API**: Fluent interface for rapid development
- **Auto-completion**: Built-in bash/zsh/fish completion generation
- **Type-safe Errors**: Structured error handling with exit codes
- **Hot-swappable Commands**: Dynamic command registration and modification

## Compatibility and Support

Orpheus is designed for Go 1.23+ environments and follows Long-Term Support guidelines to ensure consistent performance across production deployments.

## Performance

Orpheus delivers exceptional performance through elegant design:
```
BenchmarkOrpheus_FlagParsingOnly-8       2310162               511.4 ns/op            96 B/op          3 allocs/op
BenchmarkFlashFlags_Only-8               1257826               946.4 ns/op          1009 B/op         15 allocs/op
BenchmarkPflag_Only-8                     696459              1588 ns/op            1761 B/op         23 allocs/op
BenchmarkStdFlag_Only-8                  1504788               790.1 ns/op           945 B/op         13 allocs/op
```
*AMD Ryzen 5 7520U with Radeon Graphics - Startup overhead and parsing of a single command with 3 flags*

**[Complete Performance Benchmarks →](./benchmarks/benchmark_test.go)**

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

// Add subcommands using fluent API
remoteCmd.Subcommand("add", "Add a remote", func(ctx *orpheus.Context) error {
    name, url := ctx.GetArg(0), ctx.GetArg(1)
    fmt.Printf("Added remote: %s -> %s\n", name, url)
    return nil
})

remoteCmd.Subcommand("list", "List remotes", func(ctx *orpheus.Context) error {
    fmt.Println("origin\thttps://github.com/user/repo.git")
    return nil
})

app.AddCommand(remoteCmd)

// Usage: ./myapp remote add origin https://github.com/user/repo.git
//        ./myapp remote list
```

**[Complete Examples →](./docs/EXAMPLES.md)**

## The Philosophy Behind Orpheus

In Greek mythology, Orpheus was the divine musician whose lyre could move stones, tame wild beasts, and convince even the gods to change their will. His music possessed such profound harmony that it could bridge the mortal and divine realms, commanding attention through pure eloquence rather than force. When Orpheus played, complexity became simple, chaos became order, and the impossible became achievable.

Orpheus, the CLI framework, embodies this same philosophy—it transforms the complex dance between user intent and application execution into a harmonious symphony. Like the mythological bard whose music could open any door, Orpheus CLI opens the gateway between human commands and program logic with such elegance that developers forget about the underlying complexity.

Every command in Orpheus flows like a musical phrase: clear in purpose, swift in execution, and beautiful in its simplicity. Just as Orpheus's lyre could make the gods themselves pause and listen, Orpheus CLI makes even the most sophisticated applications feel intuitive and approachable.

### Core Components

- **App**: Main application container with command routing
- **Command**: Individual command with handler and flags
- **Context**: Execution context with arguments and flags
- **Errors**: Type-safe error system with exit codes
- **Completion**: Auto-completion system for multiple shells

## Testing

```bash
# Run all tests
go test ./pkg/orpheus -v

# Run with coverage
go test ./pkg/orpheus -v -cover
```

## Examples

**[Complete Examples →](./docs/EXAMPLES.md)**

Check the [demo directory](./cmd/demo) for comprehensive usage examples and the [Examples directory](./examples/) for practical implementations.

## API Reference

**[Complete API Reference →](./docs/API.md)**

## License

Orpheus is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

Orpheus • an AGILira library
