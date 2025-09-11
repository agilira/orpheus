# Orpheus Examples

This directory contains practical examples demonstrating how to use the Orpheus CLI framework.

## Examples

### [File Manager CLI](./filemanager/)

A comprehensive example showing a file management application with multiple commands, various flag types, error handling, and advanced features.

**Features demonstrated:**
- Multiple commands with different purposes
- Global and command-specific flags  
- All flag types (string, bool, int, string slice)
- Argument handling and validation
- Structured error handling with exit codes
- Help generation and shell completion
- Performance-optimized command parsing

**Commands:**
- `list` - List files and directories with filtering options
- `search` - Search files by pattern with extension filtering
- `info` - Show detailed file information
- `tree` - Display directory tree structure

## Running Examples

Each example is a self-contained Go module:

```bash
cd examples/filemanager
go mod tidy
go build -o filemanager
./filemanager --help
```

## Learning Path

1. **Start with File Manager** - Comprehensive overview of all features
2. **Study the Code** - See how to structure larger CLI applications  
3. **Experiment** - Modify examples to understand the API
4. **Build Your Own** - Create new commands and features

## Best Practices Shown

- **Error Handling**: Proper use of OrpheusError types
- **Flag Organization**: When to use global vs command flags
- **Command Structure**: Separating handlers and setup logic
- **User Experience**: Helpful error messages and verbose modes
- **Performance**: Efficient argument processing

## Next Steps

After exploring these examples:

1. Read the [main documentation](../README.md)
2. Check the [API reference](https://pkg.go.dev/github.com/agilira/orpheus)
3. Review the [benchmarks](../benchmarks/) for performance insights
4. Explore the [test cases](../pkg/orpheus/) for edge cases and usage patterns

---

Orpheus • an AGILira library