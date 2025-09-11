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
// Run the application
err := app.Run(args)
```

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

### Arguments

```go
// Get argument count
count := ctx.ArgCount()

// Get argument by index
arg := ctx.GetArg(index)
```

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

## Error Handling

### Validation Errors

```go
// Create validation error
return orpheus.ValidationError("command", "message")
```

### Custom Errors

```go
// Create custom error with exit code
return orpheus.NewOrpheusError(orpheus.ErrorExecution, "command", "message", 1)
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
                return orpheus.ValidationError("deploy", "environment argument required")
            }
            
            if env == "production" && !ctx.GetFlagBool("confirm") {
                return orpheus.NewOrpheusError(
                    orpheus.ErrorValidation, 
                    "deploy", 
                    "production deployment requires --confirm flag", 
                    2,
                )
            }
            
            // Deployment logic here
            return nil
        }).
        AddBoolFlag("confirm", "c", false, "Confirm production deployment")
    
    app.AddCommand(deployCmd)
    
    // Handle errors with proper exit codes
    if err := app.Run(os.Args[1:]); err != nil {
        if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
            log.Printf("Error: %s", orpheusErr.Error())
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