# File Manager CLI Example

This example demonstrates the core features of the Orpheus CLI framework through a simple file manager application.

## Features Demonstrated

- **Multiple Commands**: `list`, `search`, `info`, `tree`
- **Global Flags**: `--verbose`, `--config`
- **Various Flag Types**: string, bool, int, string slice
- **Argument Handling**: Required and optional arguments
- **Error Handling**: Typed errors with proper exit codes
- **Help Generation**: Automatic help for commands and flags
- **Shell Completion**: Built-in completion command

## Usage

### Build and Run

```bash
cd examples/filemanager
go mod tidy
go build -o filemanager
```

### Commands

#### List Files
```bash
# Basic listing
./filemanager list

# List with options
./filemanager list --path /tmp --all --long --limit 10

# Using global verbose flag
./filemanager --verbose list --path .
```

#### Search Files
```bash
# Search for Go files
./filemanager search --pattern "*.go"

# Search with extension filter
./filemanager search --pattern "test*" --ext go,txt

# Search in specific directory
./filemanager search --pattern "main*" --dir ./src --recursive
```

#### File Information
```bash
# Basic info
./filemanager info README.md

# Detailed info with timestamps
./filemanager info --timestamps main.go

# Info without size and permissions
./filemanager info --size=false --permissions=false file.txt
```

#### Directory Tree
```bash
# Basic tree
./filemanager tree

# Tree with options
./filemanager tree --root /tmp --depth 2 --dirs-only
```

### Help and Completion

```bash
# General help
./filemanager help

# Command-specific help
./filemanager help search

# Generate shell completion
./filemanager completion bash > completion.sh
source completion.sh
```

## Code Highlights

### Application Setup
```go
app := orpheus.New("filemanager").
    SetDescription("A simple file manager CLI").
    SetVersion("1.0.0")

// Global flags
app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output")
```

### Command Definition
```go
// Simple command with inline handler
app.Command("list", "List files and directories", listHandler).
    AddFlag("path", "p", ".", "Path to list").
    AddBoolFlag("all", "a", false, "Show hidden files")

// Complex command with builder pattern
cmd := orpheus.NewCommand("search", "Search for files").
    SetHandler(searchHandler).
    AddStringSliceFlag("ext", "e", []string{}, "File extensions").
    SetLongDescription("Search for files matching patterns...").
    AddExample("filemanager search --pattern '*.go'")

app.AddCommand(cmd)
```

### Error Handling
```go
if ctx.ArgCount() == 0 {
    return orpheus.ValidationError("info", "missing file path argument")
}

if err != nil {
    return orpheus.ExecutionError("search", fmt.Sprintf("search failed: %v", err))
}
```

### Context Usage
```go
func searchHandler(ctx *orpheus.Context) error {
    // Get flags
    pattern := ctx.GetFlagString("pattern")
    recursive := ctx.GetFlagBool("recursive")
    extensions := ctx.GetFlagStringSlice("ext")
    
    // Get global flags
    verbose := ctx.GetGlobalFlagBool("verbose")
    
    // Get arguments
    if ctx.ArgCount() > 0 {
        pattern = ctx.GetArg(0)
    }
    
    // Command logic...
}
```

## Key Features Shown

1. **Fluent API**: Chained method calls for clean setup
2. **Type Safety**: Strongly typed flag access methods
3. **Error Types**: Validation, execution, and not found errors
4. **Global State**: Global flags accessible in all commands
5. **Flexible Arguments**: Mix of flags and positional arguments
6. **Rich Help**: Long descriptions and usage examples
7. **Performance**: Fast command parsing and execution

---

Orpheus • an AGILira library