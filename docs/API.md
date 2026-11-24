# Orpheus API Reference

## Packages

| Import path | Contents |
|---|---|
| `github.com/agilira/orpheus` | App, Command, Context, help, completion |
| `github.com/agilira/orpheus/errs` | Error, the `ErrCode*` codes and their constructors |
| `github.com/agilira/orpheus/observe` | Logger, AuditLogger, Tracer, MetricsCollector, Field |
| `github.com/agilira/orpheus/storage` | Storage, Config, the plugin manager, storage errors |
| `github.com/agilira/orpheus/security` | InputValidator, ValidateSecurePath, file permission checks |
| `github.com/agilira/orpheus/prompt` | Prompter and its terminal implementation |
| `github.com/agilira/orpheus/metrics` | Prometheus and JSON Lines collectors |
| `github.com/agilira/orpheus/term` | Terminal capabilities, colors, tables, spinner |

The v1.11.0 import paths `pkg/orpheus`, `pkg/metrics` and `pkg/term` keep
the same API: their names are aliases of the types above, so code using
them compiles unchanged and values pass freely between the two paths.

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
// Run the application with Orpheus' default signal-aware context
err := app.Run(args)

// Run with caller-owned cancellation, deadlines, values, or tracing context
err := app.RunContext(ctx, args)
```

`Run` handles the common CLI lifecycle: command handlers can observe Ctrl-C and,
on POSIX platforms, SIGTERM cancellation through `ctx.Context()`. Use
`RunContext` when a caller already owns the lifecycle, such as tests, embedded
CLIs, parent processes, timeouts, or request-scoped observability.

An `App` holds its parsed flags, so `Run` and `RunContext` must not be called
concurrently on the same `App`, nor from inside one of its handlers. Calling
them one after another is fine: each run starts from the flag defaults. To
run commands in parallel, as a server executing CLI commands might, build one
`App` per goroutine. Cobra has the same rule; giving every run its own copy of
the flags would cost allocations on every dispatch.

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
// Access the execution context from App.Run or App.RunContext.
// For manually constructed Context values, this falls back to context.Background().
ctx.Context()
```

Command handlers can use `ctx.Context()` when calling context-aware APIs such as
storage providers, tracers, loggers, HTTP clients, or long-running operations.
With `Run`, this context is canceled by Orpheus' default signal handling.

### Arguments

`GetArg` and `ArgCount` return the **positional arguments**: flag tokens and
their values are excluded. For `app remote add -f origin URL`, `GetArg(0)` is
`"origin"`, wherever the flag is written.

```go
count := ctx.ArgCount()  // number of positional arguments
arg := ctx.GetArg(index) // "" when index is out of range
```

Before v1.9.0 they indexed the raw arguments, so a flag written before the
positionals was read as one. The raw arguments, flags included, are still in
the `ctx.Args` field.

`Positional()`, `PositionalCount()` and `GetPositional(index)`, added in v1.3.0
for the same purpose, return the same arguments.

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
app.SetPrompter(prompt.NewTerminal())

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

Subcommand context propagation: a subcommand handler inherits the whole
execution context — `Prompter`, `Storage`, global flags and the
`context.Context` installed by `Run`/`RunContext`. Nothing needs manual
threading (`Prompter` and `Storage` fixed in v1.3.0, `context.Context` in
v1.5.0).

### Storage

Configure a storage backend on the `App` and access it in any handler.

```go
// Wire a storage backend at application startup
app.SetStorage(myStorageBackend)

// Or configure one from a StorageConfig. This never aborts startup, so check
// why it failed when the application requires storage:
app.ConfigureStorage(config)
if err := app.StorageError(); err != nil {
    return err
}

// In a handler
store, err := ctx.RequireStorage()
if err != nil {
    return err
}

if err := store.Set(ctx, "key", []byte("value")); err != nil {
    return err
}
data, err := store.Get(ctx, "key")
keys, err := store.List(ctx, "prefix:")
if err := store.Delete(ctx, "key"); err != nil {
    return err
}
```

## Error Handling

The framework provides enhanced structured error handling with go-errors integration.

### Built-in Error Types

```go
// Create enhanced errors with automatic user messages and severity
return errs.ValidationError("deploy", "missing environment argument")
return errs.ExecutionError("deploy", "failed to connect to server")
return errs.NotFoundError("deploy", "configuration file not found")
return errs.InternalError("unexpected panic in handler")
```

### Enhanced Error Features

```go
// Create enhanced error with context and user message
err := errs.ValidationError("deploy", "missing environment").
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
if err.ErrorCode() == errs.ErrCodeValidation {
    // Handle specific error code
}
```

### Custom Error Creation

```go
// Create custom error with specific code
return errs.NewError(errs.ErrCodeExecution, "deploy", "connection failed")
```

### End-to-End Error Integration

```go
package main

import (
    "log"
    "os"
    
    "github.com/agilira/orpheus"
    "github.com/agilira/orpheus/errs"
)

func main() {
    app := orpheus.New("myapp").SetVersion("1.0.0")
    
    deployCmd := orpheus.NewCommand("deploy", "Deploy application").
        SetHandler(func(ctx *orpheus.Context) error {
            env := ctx.GetArg(0)
            if env == "" {
                return errs.ValidationError("deploy", "environment argument required").
                    WithUserMessage("Please specify a deployment environment (dev, staging, prod)").
                    WithContext("available_environments", []string{"dev", "staging", "prod"})
            }
            
            if env == "production" && !ctx.GetFlagBool("confirm") {
                return errs.ValidationError("deploy", "production deployment requires confirmation").
                    WithUserMessage("Production deployments require the --confirm flag for safety").
                    WithContext("environment", env).
                    WithSeverity("critical")
            }
            
            // Simulate deployment that might fail
            if env == "staging" {
                return errs.ExecutionError("deploy", "failed to connect to staging server").
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
        if orpheusErr, ok := err.(*errs.Error); ok {
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

## Input Validation

`InputValidator` checks values before a handler uses them; see
[SECURITY.md](SECURITY.md) for what each check accepts and rejects.

```go
validator := security.NewInputValidator(security.DefaultValidationConfig())

result := validator.ValidatePathFlag("config", path)       // *ValidatedInput
result  = validator.ValidateStringFlag("name", value)      // *ValidatedInput
result  = validator.ValidateEnvironmentValue("HOME", v)    // *ValidatedInput
result  = validator.ValidateFileOperation(path, "write")   // *ValidatedInput
err    := validator.ValidateInput(value)                   // error, warnings included
err     = validator.ValidateArgs(os.Args[1:])              // error (v1.11.0)
```

`ValidateArgs` enforces the command-line size limits of `ValidationConfig`:
each argument at most `MaxArgLength` bytes (default 4096), all arguments
together at most `MaxTotalArgsSize` bytes (default 64 KiB). Orpheus does not
call it; call it before `Run`:

```go
func main() {
    app := orpheus.New("mytool")
    // ... commands ...

    validator := security.NewInputValidator(security.DefaultValidationConfig())
    if err := validator.ValidateArgs(os.Args[1:]); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(2)
    }
    if err := app.Run(os.Args[1:]); err != nil {
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