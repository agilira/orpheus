// observability.go: observability interfaces for orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import "context"

// Logger provides structured logging capabilities for CLI applications
type Logger interface {
	// Trace logs a trace-level message with optional fields
	Trace(ctx context.Context, msg string, fields ...Field)

	// Debug logs a debug-level message with optional fields
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info logs an info-level message with optional fields
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn logs a warning-level message with optional fields
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error logs an error-level message with optional fields
	Error(ctx context.Context, msg string, fields ...Field)

	// WithFields returns a logger with additional context fields
	WithFields(fields ...Field) Logger
}

// AuditLogger provides audit trail capabilities for CLI applications
type AuditLogger interface {
	// LogCommand records command execution with metadata
	LogCommand(ctx context.Context, command string, args []string, user string, fields ...Field)

	// LogAccess records resource access attempts
	LogAccess(ctx context.Context, resource string, action string, allowed bool, fields ...Field)

	// LogSecurity records security-related events
	LogSecurity(ctx context.Context, event string, severity string, fields ...Field)

	// LogPerformance records performance metrics and timings
	LogPerformance(ctx context.Context, operation string, duration int64, fields ...Field)
}

// Tracer provides distributed tracing capabilities using OpenTelemetry patterns
type Tracer interface {
	// StartSpan creates a new span from the context
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext extracts the current span from context
	SpanFromContext(ctx context.Context) Span
}

// Span represents a single trace span
type Span interface {
	// SetAttribute adds an attribute to the span
	SetAttribute(key string, value interface{})

	// SetStatus sets the span status
	SetStatus(code StatusCode, description string)

	// RecordError records an error on the span
	RecordError(err error, opts ...ErrorOption)

	// End completes the span
	End()
}

// MetricsCollector provides metrics collection capabilities
type MetricsCollector interface {
	// Counter creates or retrieves a counter metric
	Counter(name string, description string, labels ...string) Counter

	// Gauge creates or retrieves a gauge metric
	Gauge(name string, description string, labels ...string) Gauge

	// Histogram creates or retrieves a histogram metric
	Histogram(name string, description string, buckets []float64, labels ...string) Histogram
}

// Counter represents a monotonically increasing counter
type Counter interface {
	// Inc increments the counter by 1
	Inc(ctx context.Context, labels ...string)

	// Add increments the counter by the given value
	Add(ctx context.Context, value float64, labels ...string)
}

// Gauge represents a value that can go up and down
type Gauge interface {
	// Set sets the gauge to the given value
	Set(ctx context.Context, value float64, labels ...string)

	// Inc increments the gauge by 1
	Inc(ctx context.Context, labels ...string)

	// Dec decrements the gauge by 1
	Dec(ctx context.Context, labels ...string)

	// Add adds the given value to the gauge
	Add(ctx context.Context, value float64, labels ...string)
}

// Histogram represents a distribution of values
type Histogram interface {
	// Observe records a value in the histogram
	Observe(ctx context.Context, value float64, labels ...string)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// Helper functions for creating fields
func StringField(key, value string) Field {
	return Field{Key: key, Value: value}
}

func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Float64Field(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func BoolField(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func ErrorField(err error) Field {
	return Field{Key: "error", Value: err}
}

// SpanOption configures span creation
type SpanOption interface {
	apply(*spanConfig)
}

// ErrorOption configures error recording
type ErrorOption interface {
	apply(*errorConfig)
}

// StatusCode represents span status codes
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// Internal configuration types (not exported)
type spanConfig struct {
	_ struct{} // placeholder for future use
}

type errorConfig struct {
	_ struct{} // placeholder for future use
}
