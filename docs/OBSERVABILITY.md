# Observability

Orpheus provides comprehensive observability capabilities through optional interfaces that enable structured logging, audit trails, distributed tracing, and metrics collection. The observability system is designed with zero overhead when not used and full OpenTelemetry compatibility.

## Overview

The observability framework consists of four main interfaces:

- **Logger**: Structured logging with multiple levels
- **AuditLogger**: Audit trail for compliance and security
- **Tracer**: Distributed tracing compatible with OpenTelemetry
- **MetricsCollector**: Application metrics and performance monitoring

All interfaces are optional and context-aware, using Go's `context.Context` for correlation and propagation.

## Key Features

- **Zero Overhead**: Interfaces have no performance impact when not configured (0.31 ns/op)
- **Context-Based**: Full integration with Go's context package and OpenTelemetry
- **Optional by Design**: Applications work perfectly without any observability configuration
- **Flexible Implementation**: Choose your own logging, tracing, and metrics libraries
- **Production Ready**: Designed for high-performance CLI applications

## Logger Interface

The `Logger` interface provides structured logging capabilities:

```go
type Logger interface {
    Trace(ctx context.Context, msg string, fields ...Field)
    Debug(ctx context.Context, msg string, fields ...Field)
    Info(ctx context.Context, msg string, fields ...Field)
    Warn(ctx context.Context, msg string, fields ...Field)
    Error(ctx context.Context, msg string, fields ...Field)
    WithFields(fields ...Field) Logger
}
```

### Usage Example

```go
app := orpheus.New("myapp").SetLogger(myLogger)

app.Command("process", "Process data", func(ctx *orpheus.Context) error {
    if logger := ctx.Logger(); logger != nil {
        logger.Info(context.Background(), "Processing started",
            StringField("operation", "process"),
            IntField("records", 100),
        )
    }
    return nil
})
```

### Field Helpers

Orpheus provides convenient field creation helpers:

```go
StringField("key", "value")
IntField("count", 42)
Float64Field("duration", 1.23)
BoolField("success", true)
ErrorField(err)
```

## AuditLogger Interface

The `AuditLogger` interface provides audit trail capabilities for compliance:

```go
type AuditLogger interface {
    LogCommand(ctx context.Context, command string, args []string, user string, fields ...Field)
    LogAccess(ctx context.Context, resource string, action string, allowed bool, fields ...Field)
    LogSecurity(ctx context.Context, event string, severity string, fields ...Field)
    LogPerformance(ctx context.Context, operation string, duration int64, fields ...Field)
}
```

### Usage Example

```go
app := orpheus.New("myapp").SetAuditLogger(myAuditLogger)

app.Command("deploy", "Deploy application", func(ctx *orpheus.Context) error {
    if audit := ctx.AuditLogger(); audit != nil {
        audit.LogCommand(context.Background(), "deploy", ctx.Args(), "demo-user")
        audit.LogAccess(context.Background(), "production", "deploy", true)
    }
    return nil
})
```

## Tracer Interface

The `Tracer` interface provides distributed tracing compatible with OpenTelemetry:

```go
type Tracer interface {
    StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
    SpanFromContext(ctx context.Context) Span
}

type Span interface {
    SetAttribute(key string, value interface{})
    SetStatus(code StatusCode, description string)
    RecordError(err error, opts ...ErrorOption)
    End()
}
```

### Usage Example

```go
app := orpheus.New("myapp").SetTracer(myTracer)

app.Command("backup", "Backup data", func(ctx *orpheus.Context) error {
    if tracer := ctx.Tracer(); tracer != nil {
        spanCtx, span := tracer.StartSpan(context.Background(), "backup_operation")
        defer span.End()
        
        span.SetAttribute("backup_type", "full")
        
        // Use spanCtx for downstream operations
        span.SetAttribute("backup_size", "10GB")
        span.SetStatus(StatusCodeOK, "Backup completed")
    }
    return nil
})
```

## MetricsCollector Interface

The `MetricsCollector` interface provides application metrics:

```go
type MetricsCollector interface {
    Counter(name string, description string, labels ...string) Counter
    Gauge(name string, description string, labels ...string) Gauge
    Histogram(name string, description string, buckets []float64, labels ...string) Histogram
}
```

### Usage Example

```go
app := orpheus.New("myapp").SetMetricsCollector(myMetrics)

app.Command("serve", "Start server", func(ctx *orpheus.Context) error {
    if metrics := ctx.MetricsCollector(); metrics != nil {
        requestCounter := metrics.Counter("requests_total", "Total requests", "method", "status")
        requestDuration := metrics.Histogram("request_duration_seconds", "Request duration", 
            []float64{0.1, 0.5, 1.0, 2.0, 5.0})
        
        // In request handler
        requestCounter.Inc(context.Background(), "GET", "200")
        requestDuration.Observe(context.Background(), 0.25, "GET")
    }
    return nil
})
```

## Complete Integration Example

Here's a complete example showing all observability interfaces:

```go
package main

import (
    "context"
    "time"
    "github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
    // Configure observability
    app := orpheus.New("myapp").
        SetLogger(myLogger).
        SetAuditLogger(myAuditLogger).
        SetTracer(myTracer).
        SetMetricsCollector(myMetrics)

    app.Command("process", "Process data with full observability", func(ctx *orpheus.Context) error {
        start := time.Now()
        
        // Distributed tracing
        var span orpheus.Span
        if tracer := ctx.Tracer(); tracer != nil {
            spanCtx, s := tracer.StartSpan(context.Background(), "process_data")
            span = s
            defer span.End()
            ctx = context.WithValue(ctx, "span_context", spanCtx)
        }
        
        // Structured logging
        if logger := ctx.Logger(); logger != nil {
            logger.Info(context.Background(), "Processing started",
                StringField("operation", "process"),
            )
        }
        
        // Audit logging
        if audit := ctx.AuditLogger(); audit != nil {
            audit.LogCommand(context.Background(), "process", ctx.Args(), "demo-user")
        }
        
        // Metrics
        if metrics := ctx.MetricsCollector(); metrics != nil {
            counter := metrics.Counter("commands_total", "Total commands", "command")
            counter.Inc(context.Background(), "process")
        }
        
        // Your business logic here
        time.Sleep(100 * time.Millisecond) // Simulate processing
        var err error
        
        // Record results
        duration := time.Since(start).Milliseconds()
        
        if span != nil {
            if err != nil {
                span.RecordError(err)
                span.SetStatus(StatusCodeError, "Processing failed")
            } else {
                span.SetStatus(StatusCodeOK, "Processing completed")
            }
        }
        
        if logger := ctx.Logger(); logger != nil {
            if err != nil {
                logger.Error(context.Background(), "Processing failed", ErrorField(err))
            } else {
                logger.Info(context.Background(), "Processing completed successfully",
                    IntField("duration_ms", int(duration)),
                )
            }
        }
        
        if audit := ctx.AuditLogger(); audit != nil {
            audit.LogPerformance(context.Background(), "process", duration)
        }
        
        return err
    })
    
    app.Run(os.Args[1:])
}
```

## OpenTelemetry Integration

Orpheus observability interfaces are designed to work seamlessly with OpenTelemetry:

### Context Propagation

```go
// OpenTelemetry spans automatically propagate through context
func (t *MyTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
    // Use OpenTelemetry's trace.SpanFromContext and trace.SpanContextFromContext
    return otel.Tracer("myapp").Start(ctx, name)
}
```

### Automatic Correlation

When using OpenTelemetry-compatible implementations, logs, metrics, and traces are automatically correlated through the context, providing complete observability across your CLI application.

## Performance

The observability framework is designed for zero overhead:

- **No Configuration**: 0.24 ns/op overhead (essentially zero)
- **With Configuration**: ~24 ns/op overhead for full observability
- **Context Passing**: Native Go context performance
- **Interface Calls**: Optimized for high-frequency operations

## Best Practices

1. **Always Check for nil**: Interfaces may be nil if not configured
2. **Use Context**: Pass context through your application for correlation
3. **Structured Fields**: Use field helpers for consistent logging
4. **Defer Span.End()**: Always close spans to avoid memory leaks
5. **Error Handling**: Record errors in both logs and spans
6. **Performance Metrics**: Use metrics to track application performance

## Implementation Examples

See the complete working example in `examples/observability/` for:

- Simple logger implementation
- Audit logger implementation  
- Integration patterns
- Test suite
- Performance benchmarks

The example demonstrates how to implement each interface and integrate them into a real CLI application.

---

Orpheus • an AGILira library
