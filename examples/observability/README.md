# Observability Example

This example demonstrates Orpheus's comprehensive observability features including structured logging, audit trails, distributed tracing interfaces, and metrics collection.

## Overview

The example shows how to:

- Implement simple Logger and AuditLogger interfaces
- Configure observability for a CLI application
- Use observability features in command handlers
- Create commands that work with or without observability
- Test observability implementations
- Measure performance overhead

## Running the Example

```bash
# Build the example
go build .

# Run with observability
./observability process --file data.txt

# Run without observability features
./observability simple

# Show help
./observability help
```

## Sample Output

### Process Command with Observability

```bash
$ ./observability process --file mydata.txt
[2025-09-11 21:41:17] MyApp INFO: Starting data processing operation=process user=demo-user
[2025-09-11 21:41:17] AUDIT: user=demo-user command=process args=[--file mydata.txt]
[2025-09-11 21:41:17] ACCESS: resource=data.txt action=read status=ALLOWED
[2025-09-11 21:41:17] MyApp DEBUG: Processing file filename=mydata.txt records=100
[2025-09-11 21:41:17] MyApp INFO: Processing completed successfully filename=mydata.txt duration_ms=100
[2025-09-11 21:41:17] PERFORMANCE: operation=process duration=100ms
Processed file: mydata.txt
```

### Simple Command without Observability

```bash
$ ./observability simple
This command doesn't use observability - and that's fine!
```

## Implementation Details

### SimpleLogger

The `SimpleLogger` demonstrates basic structured logging:

- Timestamped log entries
- Log levels (TRACE, DEBUG, INFO, WARN, ERROR)
- Structured fields support
- Configurable prefix

### SimpleAuditLogger

The `SimpleAuditLogger` demonstrates audit trail logging:

- Command execution logging
- Resource access logging
- Security event logging
- Performance metrics logging

### Integration Pattern

The example shows the recommended pattern for using observability:

```go
// Always check if observability is configured
if logger := ctx.Logger(); logger != nil {
    logger.Info(context.Background(), "Operation started",
        orpheus.StringField("operation", "process"),
    )
}

if audit := ctx.AuditLogger(); audit != nil {
    audit.LogCommand(context.Background(), "process", ctx.Args, "user")
}
```

## Testing

The example includes a comprehensive test suite:

```bash
# Run all tests
go test -v .

# Run benchmarks
go test -bench=. .

# Run specific test
go test -run TestObservabilityExample .
```

### Test Coverage

- **Functionality Tests**: Verify observability features work correctly
- **Integration Tests**: Test observability within command context
- **Logger Tests**: Test individual logger implementations
- **Benchmark Tests**: Measure performance overhead

### Performance Results

```
BenchmarkObservabilityOverhead/WithObservability-8      4998955    207.7 ns/op
BenchmarkObservabilityOverhead/WithoutObservability-8   6355461    183.4 ns/op
```

**Overhead**: Only 24.3 ns/op difference (~13% overhead for full observability)

## Key Features Demonstrated

### Zero Overhead Design

- Commands work perfectly without any observability configuration
- Nil checks prevent panics when interfaces aren't set
- Minimal performance impact when observability is enabled

### Context Integration

- All observability interfaces accept `context.Context`
- Compatible with OpenTelemetry's context propagation
- Enables distributed tracing across CLI operations

### Flexible Implementation

- Simple reference implementations provided
- Easy to swap with production logging libraries (logrus, zap, slog)
- Compatible with OpenTelemetry, Prometheus, Jaeger, etc.

### Production Ready

- Structured logging with fields
- Audit trails for compliance
- Performance monitoring capabilities
- Error correlation across systems

## Real-World Usage

This example can be extended for production use:

### Logger Integration

```go
// Using zap
logger := zap.NewProduction()
app.SetLogger(&ZapLoggerAdapter{logger})

// Using logrus
logger := logrus.New()
app.SetLogger(&LogrusLoggerAdapter{logger})

// Using slog (Go 1.21+)
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
app.SetLogger(&SlogLoggerAdapter{logger})
```

### Metrics Integration

```go
// Using Prometheus
registry := prometheus.NewRegistry()
app.SetMetricsCollector(&PrometheusCollector{registry})

// Using OpenTelemetry
provider := metric.NewMeterProvider()
app.SetMetricsCollector(&OTelMetricsCollector{provider})
```

### Tracing Integration

```go
// Using Jaeger
tracer := jaeger.NewTracer("myapp")
app.SetTracer(&JaegerTracerAdapter{tracer})

// Using OpenTelemetry
provider := trace.NewTracerProvider()
app.SetTracer(&OTelTracerAdapter{provider.Tracer("myapp")})
```

## Files Structure

```
examples/observability/
├── main.go           # Main application with observability
├── main_test.go      # Comprehensive test suite
├── go.mod           # Module definition
└── README.md        # This file
```

## Next Steps

1. **Extend Implementations**: Add more sophisticated logging, metrics, or tracing
2. **Add Middleware**: Create automatic observability middleware for all commands  
3. **Configuration**: Add configuration files for observability settings
4. **Integration**: Connect to real observability backends (Jaeger, Prometheus, etc.)
5. **Dashboards**: Create monitoring dashboards for your CLI applications

For complete documentation, see [docs/OBSERVABILITY.md](../../docs/OBSERVABILITY.md).
