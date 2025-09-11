# Orpheus Examples

## Advanced Features

### Custom Completion

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

### Shell Completion Setup

```bash
# Generate completion script
./myapp completion bash > /etc/bash_completion.d/myapp

# Or for zsh
./myapp completion zsh > "${fpath[1]}/_myapp"

# Or for fish
./myapp completion fish > ~/.config/fish/completions/myapp.fish
```

## Advanced Use Cases

#### Subcommands (Git-style)

Orpheus now supports native subcommands with elegant parent-child relationships:

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
    app := orpheus.New("gitlike").
        SetDescription("Git-style CLI with native subcommands").
        SetVersion("1.0.0")

    // Create parent commands
    remoteCmd := orpheus.NewCommand("remote", "Manage remote repositories")
    configCmd := orpheus.NewCommand("config", "Configuration management")
    branchCmd := orpheus.NewCommand("branch", "Branch operations")

    // Add subcommands using the native API
    remoteCmd.Subcommand("add", "Add a new remote", func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 2 {
            return orpheus.ValidationError("remote add", "requires name and URL").
                WithUserMessage("Please provide both remote name and URL").
                WithContext("expected_args", []string{"name", "url"}).
                WithContext("provided_args", ctx.ArgCount())
        }
        name, url := ctx.GetArg(0), ctx.GetArg(1)
        fmt.Printf("Added remote: %s -> %s\n", name, url)
        return nil
    })

    remoteCmd.Subcommand("remove", "Remove a remote", func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 1 {
            return orpheus.ValidationError("remote remove", "requires remote name").
                WithUserMessage("Please specify the name of the remote to remove")
        }
        name := ctx.GetArg(0)
        fmt.Printf("Removed remote: %s\n", name)
        return nil
    })

    remoteCmd.Subcommand("list", "List all remotes", func(ctx *orpheus.Context) error {
        fmt.Println("origin\thttps://github.com/user/repo.git")
        fmt.Println("upstream\thttps://github.com/upstream/repo.git")
        return nil
    })

    // Config subcommands with persistence
    configCmd.Subcommand("set", "Set configuration value", func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 2 {
            return orpheus.ValidationError("config set", "requires key and value")
        }
        key, value := ctx.GetArg(0), ctx.GetArg(1)
        // Save to config file
        fmt.Printf("Set: %s = %s\n", key, value)
        return nil
    })

    configCmd.Subcommand("get", "Get configuration value", func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 1 {
            return orpheus.ValidationError("config get", "requires key")
        }
        key := ctx.GetArg(0)
        // Load from config file
        fmt.Printf("user.name\n") // Example output
        return nil
    })

    configCmd.Subcommand("list", "List all configuration", func(ctx *orpheus.Context) error {
        fmt.Println("user.name=Developer")
        fmt.Println("user.email=dev@example.com")
        return nil
    })

    // Branch subcommands
    branchCmd.Subcommand("list", "List branches", func(ctx *orpheus.Context) error {
        fmt.Println("* main")
        fmt.Println("  develop")
        fmt.Println("  feature/new-ui")
        return nil
    })

    branchCmd.Subcommand("create", "Create branch", func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 1 {
            return orpheus.ValidationError("branch create", "requires branch name")
        }
        name := ctx.GetArg(0)
        fmt.Printf("Created branch '%s'\n", name)
        return nil
    })

    // Add commands to app
    app.AddCommand(remoteCmd)
    app.AddCommand(configCmd)
    app.AddCommand(branchCmd)

    // Usage examples:
    // ./gitlike remote add origin https://github.com/user/repo.git
    // ./gitlike config set user.name "Developer"
    // ./gitlike branch create feature/awesome
    // ./gitlike remote --help    (shows subcommands automatically)
    
    if err := app.Run(os.Args[1:]); err != nil {
        if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
            log.Printf("Error: %s", orpheusErr.UserMessage())
            if orpheusErr.IsRetryable() {
                log.Printf("This operation can be retried")
            }
            os.Exit(orpheusErr.ExitCode())
        }
        log.Fatal(err)
    }
}
```

**Key advantages of subcommands:**
- **Automatic help generation**: `./app remote --help` shows all subcommands
- **Clean command structure**: Parent-child relationships are explicit
- **Path-aware context**: Subcommands know their full command path
- **Elegant API**: No manual argument parsing or switch statements
- **Completion support**: Auto-completion works for nested commands

**Working example**: Check [`./examples/gitlike/`](./examples/gitlike/) for a complete implementation with JSON persistence.

#### Subcommand Aliases and Shortcuts

With native subcommands, you can create aliases at both command and subcommand levels:

```go
func main() {
    app := orpheus.New("myapp").
        SetDescription("CLI with subcommand aliases").
        SetVersion("1.0.0")

    // Git-style remote command with aliases
    remoteCmd := orpheus.NewCommand("remote", "Manage remotes")
    
    // Add subcommands with aliases
    listHandler := func(ctx *orpheus.Context) error {
        fmt.Println("origin\thttps://github.com/user/repo.git")
        return nil
    }
    
    // Multiple aliases for the same subcommand
    remoteCmd.Subcommand("list", "List all remotes", listHandler)
    remoteCmd.Subcommand("ls", "List all remotes (alias)", listHandler)
    remoteCmd.Subcommand("l", "List all remotes (short)", listHandler)
    
    // Remove with aliases
    removeHandler := func(ctx *orpheus.Context) error {
        if ctx.ArgCount() < 1 {
            return orpheus.ValidationError("remote remove", "requires name")
        }
        fmt.Printf("Removed remote: %s\n", ctx.GetArg(0))
        return nil
    }
    
    remoteCmd.Subcommand("remove", "Remove a remote", removeHandler)
    remoteCmd.Subcommand("rm", "Remove a remote (alias)", removeHandler)
    remoteCmd.Subcommand("delete", "Remove a remote (alias)", removeHandler)
    
    app.AddCommand(remoteCmd)
    
    // Usage examples:
    // ./myapp remote list    -> lists remotes
    // ./myapp remote ls      -> same as list (alias)
    // ./myapp remote l       -> same as list (short alias)
    // ./myapp remote remove origin
    // ./myapp remote rm origin     -> same as remove (alias)
    // ./myapp remote delete origin -> same as remove (alias)
    
    return app.Run(os.Args[1:])
}
```

#### Advanced Subcommand Patterns

Modern patterns using native subcommands:

```go
// Docker-style container management
func main() {
    app := orpheus.New("docker-like").SetVersion("1.0.0")
    
    containerCmd := orpheus.NewCommand("container", "Container operations")
    
    // Add flags to parent command that apply to all subcommands
    containerCmd.AddFlag("host", "H", "localhost", "Docker host")
    containerCmd.AddBoolFlag("tls", "", false, "Use TLS")
    
    // Subcommands inherit parent flags
    containerCmd.Subcommand("run", "Run a container", func(ctx *orpheus.Context) error {
        host := ctx.GetFlagString("host")
        image := ctx.GetArg(0)
        fmt.Printf("Running %s on %s\n", image, host)
        return nil
    })
    
    containerCmd.Subcommand("exec", "Execute in container", func(ctx *orpheus.Context) error {
        container := ctx.GetArg(0)
        command := ctx.GetArg(1)
        fmt.Printf("Executing '%s' in %s\n", command, container)
        return nil
    })
    
    containerCmd.Subcommand("logs", "Show container logs", func(ctx *orpheus.Context) error {
        container := ctx.GetArg(0)
        follow := ctx.GetFlagBool("follow")
        fmt.Printf("Showing logs for %s (follow: %v)\n", container, follow)
        return nil
    })
    
    // Add subcommand-specific flags
    logsCmd := orpheus.NewCommand("logs", "Show container logs").
        SetHandler(func(ctx *orpheus.Context) error {
            container := ctx.GetArg(0)
            follow := ctx.GetFlagBool("follow")
            lines := ctx.GetFlagInt("tail")
            fmt.Printf("Logs for %s (follow: %v, lines: %d)\n", container, follow, lines)
            return nil
        })
    logsCmd.AddBoolFlag("follow", "f", false, "Follow log output")
    logsCmd.AddIntFlag("tail", "t", 100, "Number of lines to show")
    
    containerCmd.AddCommand(logsCmd)
    app.AddCommand(containerCmd)
    
    // Usage:
    // ./app container --host remote.docker.com run nginx
    // ./app container logs --follow --tail 50 mycontainer
    
    return app.Run(os.Args[1:])
}

// Kubernetes-style resource management
func setupK8sStyle() *orpheus.App {
    app := orpheus.New("kubectl-like").SetVersion("1.0.0")
    
    // Global flags
    app.AddGlobalFlag("namespace", "n", "default", "Kubernetes namespace")
    app.AddGlobalFlag("kubeconfig", "", "", "Path to kubeconfig file")
    
    // Get command with resource types as subcommands
    getCmd := orpheus.NewCommand("get", "Display resources")
    getCmd.Subcommand("pods", "List pods", func(ctx *orpheus.Context) error {
        ns := ctx.GetGlobalFlagString("namespace")
        fmt.Printf("Listing pods in namespace: %s\n", ns)
        return nil
    })
    getCmd.Subcommand("services", "List services", func(ctx *orpheus.Context) error {
        ns := ctx.GetGlobalFlagString("namespace")
        fmt.Printf("Listing services in namespace: %s\n", ns)
        return nil
    })
    
    app.AddCommand(getCmd)
    return app
}
```
---

Orpheus • an AGILira library