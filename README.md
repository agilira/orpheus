# Orpheus: Blazing-Fast zero-dependency CLI framework for go
### an AGILira library

[![CI/CD Pipeline](https://github.com/agilira/orpheus/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/orpheus/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/orpheus)](https://goreportcard.com/report/github.com/agilira/orpheus)
[![Coverage](https://codecov.io/gh/agilira/orpheus/branch/main/graph/badge.svg)](https://codecov.io/gh/agilira/orpheus)

Orpheus is a high-performance CLI framework designed to be super simple and **7-53x faster** than popular alternatives with zero external dependencies. Orpheus provides a simple interface to create modern, fast CLI apps similar to git.

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

## Features

- **Zero Dependencies**: Zero external dependencies for maximum portability
- **Clean API**: Fluent interface for rapid development
- **Auto-completion**: Built-in bash/zsh/fish completion generation
- **Type-safe Errors**: Structured error handling with exit codes
- **Hot-swappable Commands**: Dynamic command registration and modification

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

### Advanced Features

#### Custom Completion

```go
deployCmd := orpheus.NewCommand("deploy", "Deploy to environment").
    SetHandler(func(ctx *orpheus.Context) error {
        env := ctx.GetArg(0)
        fmt.Printf("Deploying to %s...\n", env)
        return nil
    }).
    SetCompletionHandler(func(req *orpheus.CompletionRequest) *orpheus.CompletionResult {
        if req.Type == orpheus.CompletionArgs && req.Position == 0 {
            return &orpheus.CompletionResult{
                Suggestions: []string{"production", "staging", "development"},
            }
        }
        return &orpheus.CompletionResult{Suggestions: []string{}}
    })

app.AddCommand(deployCmd)
```

#### Shell Completion Setup

```bash
# Generate completion script
./myapp completion bash > /etc/bash_completion.d/myapp

# Or for zsh
./myapp completion zsh > "${fpath[1]}/_myapp"

# Or for fish
./myapp completion fish > ~/.config/fish/completions/myapp.fish
```

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

Check the [examples directory](./cmd/demo) for comprehensive usage examples:

- **Basic CLI**: Simple command-line application
- **Advanced Completion**: Custom completion handlers
- **Error Handling**: Structured error management
- **Global Flags**: Application-wide configuration

## API Reference

### App Methods

```go
// Creation and configuration
app := orpheus.New("myapp")
app.SetDescription("Description")
app.SetVersion("1.0.0")

// Command management
app.Command("name", "description", handler)
app.AddCommand(cmd)
app.SetDefaultCommand("name")

// Global flags
app.AddGlobalFlag("name", "n", "default", "description")
app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose")

// Execution
err := app.Run(args)
```

### Command Methods

```go
// Creation and configuration
cmd := orpheus.NewCommand("name", "description")
cmd.SetHandler(handler)
cmd.SetUsage("custom usage")
cmd.SetCompletionHandler(completionHandler)

// Flags
cmd.AddFlag("name", "n", "default", "description")
cmd.AddBoolFlag("enabled", "e", false, "Enable feature")
cmd.AddIntFlag("count", "c", 10, "Count value")
```

### Context Methods

```go
// Arguments
count := ctx.ArgCount()
arg := ctx.GetArg(index)

// Flags (when flash-flags integration is complete)
value := ctx.GetFlagString("name")
enabled := ctx.GetFlagBool("enabled")
count := ctx.GetFlagInt("count")

// Global flags
verbose := ctx.GetGlobalFlagBool("verbose")
```

## License

Orpheus is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

Orpheus • an AGILira library
