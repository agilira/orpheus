# Orpheus API Reference

## App Methods

### Creation and Configuration

```go
// Create a new application
app := orpheus.New("myapp")

// Set application metadata
app.SetDescription("Description")
app.SetVersion("1.0.0")
```

### Command Management

```go
// Add commands using fluent interface
app.Command("name", "description", handler)

// Add pre-configured commands
app.AddCommand(cmd)

// Set default command when no command is specified
app.SetDefaultCommand("name")
```

### Global Flags

```go
// String flags
app.AddGlobalFlag("name", "n", "default", "description")

// Boolean flags
app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose")

// Integer flags
app.AddGlobalIntFlag("count", "c", 10, "Count value")
```

### Execution

```go
// Run the application with a background context
err := app.Run(args)

// Run with caller-owned cancellation, deadlines, values, or tracing context
err := app.RunContext(ctx, args)
```

`Run` preserves the simple path and uses `context.Background()` internally.
Use `RunContext` when your CLI should react to cancellation from signals,
timeouts, parent processes, or request-scoped observability.

## Command Methods

### Creation and Configuration

```go
// Create a new command
cmd := orpheus.NewCommand("name", "description")

// Set handler function
cmd.SetHandler(handler)

// Set custom usage string
cmd.SetUsage("custom usage")

// Set completion handler
cmd.SetCompletionHandler(completionHandler)
```

### Subcommands

```go
// Add subcommands using fluent interface
cmd.Subcommand("subcmd", "description", handler)

// Add pre-configured subcommands
subCmd := orpheus.NewCommand("subcmd", "description")
cmd.AddSubcommand(subCmd)
```

### Flags

```go
// String flags
cmd.AddFlag("name", "n", "default", "description")

// Boolean flags
cmd.AddBoolFlag("enabled", "e", false, "Enable feature")

// Integer flags
cmd.AddIntFlag("count", "c", 10, "Count value")

// Float flags
cmd.AddFloat64Flag("ratio", "r", 1.0, "Ratio value")

// String slice flags 
cmd.AddStringSliceFlag("tags", "t", []string{}, "Tags")
```

## Context Methods

### Execution Context

```go
// Access the parent context passed through App.RunContext.
// For manually constructed Context values, this falls back to context.Background().
ctx.Context()
```

Command handlers can use `ctx.Context()` when calling context-aware APIs such as
storage providers, tracers, loggers, HTTP clients, or long-running operations.

### Arguments

The `Args` surface returns the **raw arguments** passed to the command after the
command name. Flag tokens (`--name`) and their values are included in this slice.
Use this API when you rely on the documented raw-input semantics.

```go
// Get raw argument count (includes flag tokens)
count := ctx.ArgCount()

// Get raw argument by index
arg := ctx.GetArg(index)
```

### Positional Arguments

Added in v1.3.0. These accessors return the **post-parse positional arguments**:
flag tokens and their values are excluded by construction. Use this API when you
need to validate that the user supplied positional values independently of any
flags that may also be present.

```go
// Full positional slice (flag tokens excluded)
positionals := ctx.Positional()

// Number of positional arguments
count := ctx.PositionalCount()

// Safe single-index accessor — returns "" when index is out of range
value := ctx.GetPositional(index)
```

All three methods return safe zero values when `ctx.Flags` is nil (e.g. tests
building a `Context` outside the dispatch chain).

**Choosing between `Args` and `Positional`:** if your handler receives
`app cmd --flag value positional`, `ctx.GetArg(0)` returns `"--flag"` while
`ctx.GetPositional(0)` returns `"positional"`. A required-positional check
using `ArgCount() < 1` silently accepts the call in the former case; use
`PositionalCount() < 1` to express the intent correctly.

### Command Flags

```go
// Get flag values
value := ctx.GetFlagString("name")
enabled := ctx.GetFlagBool("enabled")
count := ctx.GetFlagInt("count")
ratio := ctx.GetFlagFloat64("ratio")
tags := ctx.GetFlagStringSlice("tags")

// Check if flag was set
changed := ctx.FlagChanged("name")
```

### Global Flags

```go
// Get global flag values
verbose := ctx.GetGlobalFlagBool("verbose")
namespace := ctx.GetGlobalFlagString("namespace")

// Check if global flag was set
changed := ctx.GlobalFlagChanged("verbose")
```

### Interactive Prompts (v1.2.0)

Configure a `Prompter` on the `App` and retrieve it in any handler.
Use `NewTerminalPrompter()` for real terminal interaction; inject a mock
implementation in tests for deterministic execution.

```go
// Wire a terminal prompter at application startup
app.SetPrompter(orpheus.NewTerminalPrompter())

// In a handler — get the prompter (returns error if not configured)
p, err := ctx.RequirePrompter()
if err != nil {
    return err
}

name, err := p.Ask("Your name", "anonymous")   // free-text input
pass, err := p.AskSecret("Password")            // masked terminal input
idx,  err := p.Choose("Environment", []string{"dev", "staging", "prod"}) // 0-based
ok,   err := p.Confirm("Deploy now?", false)    // yes/no
```

| Method | Purpose | Return |
|---|---|---|
| `Ask(prompt, default)` | Free-text input with optional default | `(string, error)` |
| `AskSecret(prompt)` | Masked password/key entry via `golang.org/x/term` | `(string, error)` |
| `Choose(prompt, options)` | Numbered menu selection | `(int, error)` (0-based) |
| `Confirm(prompt, defaultYes)` | Yes/no question | `(bool, error)` |

Subcommand context propagation: `Prompter` and `Storage` are automatically
propagated to subcommand handlers — no manual threading required (fixed v1.3.0).

### Storage

Configure a storage backend on the `App` and access it in any handler.

```go
// Wire a storage backend at application startup
app.SetStorage(myStorageBackend)

// In a handler
storage, err := ctx.RequireStorage()
if err != nil {
    return err
}

if err := storage.Set(ctx, "key", []byte("value")); err != nil {
    return err
}
data, err := storage.Get(ctx, "key")
keys, err := storage.List(ctx, "prefix:")
if err := storage.Delete(ctx, "key"); err != nil {
    return err
}
```

## Error Handling

The framework provides enhanced structured error handling with go-errors integration.

### Built-in Error Types

```go
// Create enhanced errors with automatic user messages and severity
return orpheus.ValidationError("deploy", "missing environment argument")
return orpheus.ExecutionError("deploy", "failed to connect to server")
return orpheus.NotFoundError("deploy", "configuration file not found")
return orpheus.InternalError("unexpected panic in handler")
```

### Enhanced Error Features

```go
// Create enhanced error with context and user message
err := orpheus.ValidationError("deploy", "missing environment").
    WithUserMessage("Please specify a deployment environment").
    WithContext("available_envs", []string{"dev", "staging", "prod"}).
    WithContext("user_input", userInput).
    AsRetryable().
    WithSeverity("warning")

// Access enhanced error information
userMsg := err.UserMessage()           // User-friendly message
isRetryable := err.IsRetryable()       // Check if operation can be retried
errorCode := err.ErrorCode()           // Get structured error code (ORF1000, etc.)
```

### Error Code Constants

```go
// Orpheus error codes for structured error handling
const (
    ErrCodeValidation = "ORF1000" // Input validation errors
    ErrCodeExecution  = "ORF1001" // Command execution errors  
    ErrCodeNotFound   = "ORF1002" // Resource not found errors
    ErrCodeInternal   = "ORF1003" // Internal framework errors
)

// Type-safe error checking
if err.IsValidationError() {
    // Handle validation errors
}
if err.ErrorCode() == orpheus.ErrCodeValidation {
    // Handle specific error code
}
```

### Custom Error Creation

```go
// Create custom error with specific code
return orpheus.NewOrpheusError(orpheus.ErrCodeExecution, "deploy", "connection failed")
```

### End-to-End Error Integration

```go
package main

import (
    "log"
    "os"
    
    "github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
    app := orpheus.New("myapp").SetVersion("1.0.0")
    
    deployCmd := orpheus.NewCommand("deploy", "Deploy application").
        SetHandler(func(ctx *orpheus.Context) error {
            env := ctx.GetArg(0)
            if env == "" {
                return orpheus.ValidationError("deploy", "environment argument required").
                    WithUserMessage("Please specify a deployment environment (dev, staging, prod)").
                    WithContext("available_environments", []string{"dev", "staging", "prod"})
            }
            
            if env == "production" && !ctx.GetFlagBool("confirm") {
                return orpheus.ValidationError("deploy", "production deployment requires confirmation").
                    WithUserMessage("Production deployments require the --confirm flag for safety").
                    WithContext("environment", env).
                    WithSeverity("critical")
            }
            
            // Simulate deployment that might fail
            if env == "staging" {
                return orpheus.ExecutionError("deploy", "failed to connect to staging server").
                    WithUserMessage("Unable to connect to the staging environment").
                    WithContext("server", "staging.example.com").
                    WithContext("port", 22).
                    AsRetryable()
            }
            
            // Deployment logic here
            return nil
        }).
        AddBoolFlag("confirm", "c", false, "Confirm production deployment")
    
    app.AddCommand(deployCmd)
    
    // Handle errors with enhanced error information
    if err := app.Run(os.Args[1:]); err != nil {
        if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
            // Display user-friendly message
            log.Printf("Error: %s", orpheusErr.UserMessage())
            
            // Log technical details for debugging
            log.Printf("Technical details: %s", orpheusErr.Error())
            log.Printf("Error code: %s", orpheusErr.ErrorCode())
            
            // Handle retryable errors
            if orpheusErr.IsRetryable() {
                log.Printf("This operation can be retried")
            }
            
            os.Exit(orpheusErr.ExitCode())
        }
        log.Fatal(err)
    }
}
```

## Completion System

### Completion Request

```go
type CompletionRequest struct {
    Type        CompletionType  // Type of completion
    CurrentWord string          // Current word being completed
    Command     string          // Command being completed
    Args        []string        // Arguments so far
    Position    int            // Position in command line
}
```

### Completion Result

```go
type CompletionResult struct {
    Suggestions []string           // List of suggestions
    Directive   CompletionDirective // Shell directive
}
```

### Custom Completion Handler

```go
cmd.SetCompletionHandler(func(req *orpheus.CompletionRequest) *orpheus.CompletionResult {
    if req.Type == orpheus.CompletionArgs && req.Position == 0 {
        return &orpheus.CompletionResult{
            Suggestions: []string{"option1", "option2", "option3"},
        }
    }
    return &orpheus.CompletionResult{Suggestions: []string{}}
})
```
---

Orpheus • an AGILira library