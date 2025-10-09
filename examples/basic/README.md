# Basic Orpheus CLI Example

This example demonstrates the fundamental features of the Orpheus CLI framework.

## Features Demonstrated

- **Simple Command Creation**: Easy command registration with handlers
- **Global Flags**: Framework-wide flags that work with all commands
- **Argument Handling**: Processing command-line arguments
- **Error Handling**: Structured error responses with validation
- **Command Completion**: Auto-completion for shell environments
- **Help Generation**: Automatic help text generation

## Usage

Build and run the example:

```bash
go build -o basic .
./basic --help
```

### Available Commands

- `greet [name]` - Greet someone (default: World)
- `echo <args...>` - Echo back the provided arguments
- `deploy <environment>` - Deploy to a specific environment
- `completion <shell>` - Generate shell completion scripts

### Examples

```bash
# Basic greeting
./basic greet
./basic greet Alice

# With verbose output
./basic --verbose greet Bob

# Echo command
./basic echo hello world

# Deploy with completion
./basic deploy production
./basic deploy staging

# Generate completions
./basic completion bash > basic_completion.bash
```

### Global Flags

- `--verbose, -v` - Enable verbose output
- `--config, -c` - Specify configuration file path
- `--help, -h` - Show help information
- `--version` - Show version information

## Testing

Run the comprehensive test suite:

```bash
go test -v
go test -cover
```

The example includes:
- Unit tests for all commands
- Error handling tests
- Global flag tests
- Help generation tests
- Completion tests
- Benchmark tests

## Code Structure

- `main.go` - Application setup and command definitions

This example serves as a template for building CLI applications with Orpheus.

---

Orpheus • an AGILira library