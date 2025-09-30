// observability_test.go: tests to verify that observability interfaces work correctly
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"testing"
)

// TestObservabilityInterfaces verifies that observability interfaces work correctly
func TestObservabilityInterfaces(t *testing.T) {
	app := New("testapp").
		SetDescription("Test app with observability").
		SetVersion("1.0.0")

	// Test that nil interfaces don't panic
	ctx := &Context{App: app}

	// These should all return nil without panicking
	if ctx.Logger() != nil {
		t.Error("Expected nil logger")
	}
	if ctx.AuditLogger() != nil {
		t.Error("Expected nil audit logger")
	}
	if ctx.Tracer() != nil {
		t.Error("Expected nil tracer")
	}
	if ctx.MetricsCollector() != nil {
		t.Error("Expected nil metrics collector")
	}
}

// TestObservabilityConfiguration verifies that observability can be configured
func TestObservabilityConfiguration(t *testing.T) {
	mockLogger := &mockLogger{}
	mockAudit := &mockAuditLogger{}
	mockTracer := &mockTracer{}
	mockMetrics := &mockMetricsCollector{}

	app := New("testapp").
		SetLogger(mockLogger).
		SetAuditLogger(mockAudit).
		SetTracer(mockTracer).
		SetMetricsCollector(mockMetrics)

	ctx := &Context{App: app}

	// Verify all components are accessible
	if ctx.Logger() != mockLogger {
		t.Error("Logger not properly set")
	}
	if ctx.AuditLogger() != mockAudit {
		t.Error("Audit logger not properly set")
	}
	if ctx.Tracer() != mockTracer {
		t.Error("Tracer not properly set")
	}
	if ctx.MetricsCollector() != mockMetrics {
		t.Error("Metrics collector not properly set")
	}
}

// Benchmark to verify zero overhead when observability is not used
func BenchmarkContextWithoutObservability(b *testing.B) {
	app := New("benchapp")
	ctx := &Context{App: app}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access observability components (should be nil checks only)
		_ = ctx.Logger()
		_ = ctx.AuditLogger()
		_ = ctx.Tracer()
		_ = ctx.MetricsCollector()
	}
}

// Mock implementations for testing
type mockLogger struct{}

func (m *mockLogger) Trace(ctx context.Context, msg string, fields ...Field) {}
func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...Field) {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...Field)  {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...Field)  {}
func (m *mockLogger) Error(ctx context.Context, msg string, fields ...Field) {}
func (m *mockLogger) WithFields(fields ...Field) Logger                      { return m }

type mockAuditLogger struct{}

func (m *mockAuditLogger) LogCommand(ctx context.Context, command string, args []string, user string, fields ...Field) {
}
func (m *mockAuditLogger) LogAccess(ctx context.Context, resource string, action string, allowed bool, fields ...Field) {
}
func (m *mockAuditLogger) LogSecurity(ctx context.Context, event string, severity string, fields ...Field) {
}
func (m *mockAuditLogger) LogPerformance(ctx context.Context, operation string, duration int64, fields ...Field) {
}

type mockTracer struct{}

func (m *mockTracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &mockSpan{}
}
func (m *mockTracer) SpanFromContext(ctx context.Context) Span { return &mockSpan{} }

type mockSpan struct{}

func (m *mockSpan) SetAttribute(key string, value interface{})    {}
func (m *mockSpan) SetStatus(code StatusCode, description string) {}
func (m *mockSpan) RecordError(err error, opts ...ErrorOption)    {}
func (m *mockSpan) End()                                          {}

type mockMetricsCollector struct{}

func (m *mockMetricsCollector) Counter(name string, description string, labels ...string) Counter {
	return &mockCounter{}
}
func (m *mockMetricsCollector) Gauge(name string, description string, labels ...string) Gauge {
	return &mockGauge{}
}
func (m *mockMetricsCollector) Histogram(name string, description string, buckets []float64, labels ...string) Histogram {
	return &mockHistogram{}
}

type mockCounter struct{}

func (m *mockCounter) Inc(ctx context.Context, labels ...string)                {}
func (m *mockCounter) Add(ctx context.Context, value float64, labels ...string) {}

type mockGauge struct{}

func (m *mockGauge) Set(ctx context.Context, value float64, labels ...string) {}
func (m *mockGauge) Inc(ctx context.Context, labels ...string)                {}
func (m *mockGauge) Dec(ctx context.Context, labels ...string)                {}
func (m *mockGauge) Add(ctx context.Context, value float64, labels ...string) {}

type mockHistogram struct{}

func (m *mockHistogram) Observe(ctx context.Context, value float64, labels ...string) {}

// TestFieldCreationHelpers verifies that field creation helper functions work correctly
func TestFieldCreationHelpers(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() Field
		expected interface{}
	}{
		{
			name: "StringFieldCreation",
			testFunc: func() Field {
				return StringField("key", "value")
			},
			expected: "value",
		},
		{
			name: "IntFieldCreation",
			testFunc: func() Field {
				return IntField("key", 42)
			},
			expected: 42,
		},
		{
			name: "Float64FieldCreation",
			testFunc: func() Field {
				return Float64Field("key", 3.14)
			},
			expected: 3.14,
		},
		{
			name: "BoolFieldCreation",
			testFunc: func() Field {
				return BoolField("key", true)
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.testFunc()
			if field.Key != "key" {
				t.Errorf("Expected field key 'key', got '%s'", field.Key)
			}
			if field.Value != tt.expected {
				t.Errorf("Expected field value %v, got %v", tt.expected, field.Value)
			}
		})
	}
}

// TestErrorFieldCreation verifies error field creation with proper error handling
func TestErrorFieldCreation(t *testing.T) {
	testError := ValidationError("test", "test error message")

	field := ErrorField(testError)

	if field.Key != "error" {
		t.Errorf("Expected field key 'error', got '%s'", field.Key)
	}

	if field.Value != testError {
		t.Errorf("Expected field value to be the error, got %v", field.Value)
	}

	// Test with nil error - ErrorField should handle this gracefully
	nilField := ErrorField(nil)
	if nilField.Key != "error" {
		t.Errorf("Expected field key 'error' even with nil error, got '%s'", nilField.Key)
	}
	if nilField.Value != nil {
		t.Errorf("Expected nil error field value, got %v", nilField.Value)
	}
}
