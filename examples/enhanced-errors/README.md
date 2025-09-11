# Enhanced Error Handling Example

This example demonstrates Orpheus CLI framework's enhanced error handling capabilities using the go-errors integration. It showcases structured error reporting, context information, user-friendly messages, and retry semantics.

## Features Demonstrated

- **Structured Error Codes**: ORF1000-ORF1003 error classification system
- **User-Friendly Messages**: Clear, actionable error messages for end users
- **Technical Details**: Detailed debugging information in development mode
- **Context Information**: Rich metadata attached to errors for troubleshooting
- **Retry Semantics**: Automatic identification of retryable operations
- **Severity Levels**: Warning, error, and critical error classification

## Usage

Build the example:

```bash
go build -o enhanced-errors main.go
```

### Basic Commands

Test validation errors with missing parameters:
```bash
./enhanced-errors validate
```

Test validation errors with invalid data:
```bash
./enhanced-errors validate --data invalid
```

Test successful validation:
```bash
./enhanced-errors validate --data "valid-content" --format json
```

### Connection Errors

Test connection timeout (retryable error):
```bash
./enhanced-errors connect --timeout 5
```

Test successful connection:
```bash
./enhanced-errors connect --timeout 30
```

### File Operations

Test file not found error:
```bash
./enhanced-errors process --file nonexistent.txt
```

Test file creation:
```bash
./enhanced-errors process --file nonexistent.txt --create
```

### Critical Errors

Test critical system error simulation:
```bash
./enhanced-errors critical --simulate
```

## Debug Mode

Enable detailed technical information by setting the DEBUG environment variable:

```bash
DEBUG=1 ./enhanced-errors validate --data invalid
```

This displays:
- Error codes (ORF1000-ORF1003)
- Technical error messages
- Command context
- Stack traces (when available)

## Error Types

### Validation Errors (ORF1000)
- Exit code: 1
- Triggered by: Invalid input, missing required parameters
- Retryable: No
- Severity: Warning

### Execution Errors (ORF1001)
- Exit code: 1
- Triggered by: Command execution failures, timeouts
- Retryable: Often yes
- Severity: Error

### Not Found Errors (ORF1002)
- Exit code: 1
- Triggered by: Missing files, unknown commands
- Retryable: No
- Severity: Warning

### Internal Errors (ORF1003)
- Exit code: 2
- Triggered by: System failures, resource exhaustion
- Retryable: Rarely
- Severity: Critical

## Error Context

Each error includes relevant context information:

- Command name and parameters
- Suggested alternatives or fixes
- System state information
- Resource availability
- Configuration details

## Integration Notes

This example uses:
- `github.com/agilira/go-errors` for structured error handling
- Orpheus CLI framework's enhanced error types
- Fluent API for error construction
- Chain-compatible error methods

## Implementation Details

The error handling follows these patterns:

1. **Error Creation**: Use specific error constructors (ValidationError, ExecutionError, etc.)
2. **Message Enhancement**: Add user-friendly messages with WithUserMessage()
3. **Context Addition**: Include relevant metadata with WithContext()
4. **Retry Configuration**: Mark retryable operations with AsRetryable()
5. **Severity Setting**: Classify error importance with WithSeverity()

## Development

To modify this example:

1. Edit `main.go` to add new commands or error scenarios
2. Run `go mod tidy` to update dependencies
3. Build and test with various input combinations
4. Use DEBUG=1 to verify error details during development

---

Orpheus • an AGILira library