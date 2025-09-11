package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// TestObservabilityExample tests the observability example functionality
func TestObservabilityExample(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "process command with default file",
			args: []string{"process"},
			expectedOutput: []string{
				"INFO: Starting data processing",
				"AUDIT: user=demo-user command=process",
				"ACCESS: resource=data.txt action=read status=ALLOWED",
				"DEBUG: Processing file",
				"INFO: Processing completed successfully",
				"PERFORMANCE: operation=process",
				"Processed file: default.txt",
			},
			expectError: false,
		},
		{
			name: "process command with custom file",
			args: []string{"process", "--file", "custom.txt"},
			expectedOutput: []string{
				"INFO: Starting data processing",
				"AUDIT: user=demo-user command=process",
				"ACCESS: resource=data.txt action=read status=ALLOWED",
				"DEBUG: Processing file filename=custom.txt",
				"INFO: Processing completed successfully filename=custom.txt",
				"PERFORMANCE: operation=process",
				"Processed file: custom.txt",
			},
			expectError: false,
		},
		{
			name: "simple command without observability",
			args: []string{"simple"},
			expectedOutput: []string{
				"This command doesn't use observability - and that's fine!",
			},
			expectError: false,
		},
		{
			name: "help command",
			args: []string{"help"},
			expectedOutput: []string{
				"observability-demo",
				"Demonstration of Orpheus observability features",
				"Available Commands:",
				"process",
				"simple",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr

			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()

			os.Stdout = wOut
			os.Stderr = wErr

			// Create buffers to capture output
			var stdout, stderr bytes.Buffer

			done := make(chan bool, 2)

			go func() {
				_, _ = stdout.ReadFrom(rOut)
				done <- true
			}()

			go func() {
				_, _ = stderr.ReadFrom(rErr)
				done <- true
			}()

			// Run the command
			err := runApp(tt.args)

			// Close pipes and restore stdout/stderr
			wOut.Close()
			wErr.Close()
			<-done
			<-done

			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			output := stdout.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestSimpleLogger tests the SimpleLogger implementation
func TestSimpleLogger(t *testing.T) {
	logger := NewSimpleLogger("TEST")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		_, _ = buf.ReadFrom(r)
		done <- true
	}()

	// Test different log levels
	ctx := context.Background()
	logger.Info(ctx, "test message", orpheus.StringField("key", "value"))
	logger.Error(ctx, "error message", orpheus.IntField("code", 500))

	w.Close()
	<-done
	os.Stdout = oldStdout

	output := buf.String()

	// Verify log format
	if !strings.Contains(output, "TEST INFO: test message key=value") {
		t.Errorf("Expected info log format, got: %s", output)
	}
	if !strings.Contains(output, "TEST ERROR: error message code=500") {
		t.Errorf("Expected error log format, got: %s", output)
	}
}

// TestSimpleAuditLogger tests the SimpleAuditLogger implementation
func TestSimpleAuditLogger(t *testing.T) {
	auditLogger := NewSimpleAuditLogger()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		_, _ = buf.ReadFrom(r)
		done <- true
	}()

	// Test audit logging
	ctx := context.Background()
	auditLogger.LogCommand(ctx, "test", []string{"arg1"}, "testuser")
	auditLogger.LogAccess(ctx, "resource1", "read", true)
	auditLogger.LogSecurity(ctx, "login_attempt", "high")
	auditLogger.LogPerformance(ctx, "operation1", 150)

	w.Close()
	<-done
	os.Stdout = oldStdout

	output := buf.String()

	// Verify audit log formats
	expectedPatterns := []string{
		"AUDIT: user=testuser command=test args=[arg1]",
		"ACCESS: resource=resource1 action=read status=ALLOWED",
		"SECURITY: event=login_attempt severity=high",
		"PERFORMANCE: operation=operation1 duration=150ms",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected audit log to contain %q, got: %s", pattern, output)
		}
	}
}

// TestObservabilityIntegration tests the integration between observability and commands
func TestObservabilityIntegration(t *testing.T) {
	logger := NewSimpleLogger("INTEGRATION")
	auditLogger := NewSimpleAuditLogger()

	app := orpheus.New("integration-test").
		SetLogger(logger).
		SetAuditLogger(auditLogger)

	// Track if observability was called
	observabilityCalled := false

	cmd := orpheus.NewCommand("test", "Test command").
		SetHandler(func(ctx *orpheus.Context) error {
			if ctx.Logger() != nil && ctx.AuditLogger() != nil {
				observabilityCalled = true

				// Test context methods
				ctx.Logger().Info(context.Background(), "test log")
				ctx.AuditLogger().LogCommand(context.Background(), "test", []string{}, "testuser")
			}
			return nil
		})

	app.AddCommand(cmd)

	err := app.Run([]string{"test"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !observabilityCalled {
		t.Error("Observability was not accessible from command context")
	}
}

// BenchmarkObservabilityOverhead benchmarks the performance overhead of observability
func BenchmarkObservabilityOverhead(b *testing.B) {
	b.Run("WithObservability", func(b *testing.B) {
		logger := NewSimpleLogger("BENCH")
		auditLogger := NewSimpleAuditLogger()

		app := orpheus.New("bench-test").
			SetLogger(logger).
			SetAuditLogger(auditLogger)

		cmd := orpheus.NewCommand("test", "Test").
			SetHandler(func(ctx *orpheus.Context) error {
				_ = ctx.Logger()
				_ = ctx.AuditLogger()
				return nil
			})

		app.AddCommand(cmd)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = app.Run([]string{"test"})
		}
	})

	b.Run("WithoutObservability", func(b *testing.B) {
		app := orpheus.New("bench-test")

		cmd := orpheus.NewCommand("test", "Test").
			SetHandler(func(ctx *orpheus.Context) error {
				_ = ctx.Logger()
				_ = ctx.AuditLogger()
				return nil
			})

		app.AddCommand(cmd)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = app.Run([]string{"test"})
		}
	})
}

// runApp is a helper function that creates and runs the application
func runApp(args []string) error {
	// Create logger and audit logger
	logger := NewSimpleLogger("MyApp")
	auditLogger := NewSimpleAuditLogger()

	// Create application with observability
	app := orpheus.New("observability-demo").
		SetDescription("Demonstration of Orpheus observability features").
		SetVersion("1.0.0").
		SetLogger(logger).
		SetAuditLogger(auditLogger)

	// Create process command
	processCmd := orpheus.NewCommand("process", "Process some data with observability").
		SetHandler(func(ctx *orpheus.Context) error {
			start := time.Now()

			// Log command start
			if logger := ctx.Logger(); logger != nil {
				logger.Info(context.Background(), "Starting data processing",
					orpheus.StringField("operation", "process"),
					orpheus.StringField("user", "demo-user"),
				)
			}

			// Log audit trail
			if audit := ctx.AuditLogger(); audit != nil {
				audit.LogCommand(context.Background(), "process", ctx.Args, "demo-user")
				audit.LogAccess(context.Background(), "data.txt", "read", true)
			}

			// Simulate some work
			fileName := ctx.GetFlagString("file")
			if fileName == "" {
				fileName = "default.txt"
			}

			// Log progress
			if logger := ctx.Logger(); logger != nil {
				logger.Debug(context.Background(), "Processing file",
					orpheus.StringField("filename", fileName),
					orpheus.IntField("records", 100),
				)
			}

			// Simulate processing time (reduced for testing)
			time.Sleep(1 * time.Millisecond)

			// Log completion
			duration := time.Since(start).Milliseconds()
			if logger := ctx.Logger(); logger != nil {
				logger.Info(context.Background(), "Processing completed successfully",
					orpheus.StringField("filename", fileName),
					orpheus.IntField("duration_ms", int(duration)),
				)
			}

			if audit := ctx.AuditLogger(); audit != nil {
				audit.LogPerformance(context.Background(), "process", duration)
			}

			fmt.Printf("Processed file: %s\n", fileName)
			return nil
		}).
		AddFlag("file", "f", "", "File to process")

	app.AddCommand(processCmd)

	// Add simple command
	simpleCmd := orpheus.NewCommand("simple", "Simple command without observability").
		SetHandler(func(ctx *orpheus.Context) error {
			fmt.Println("This command doesn't use observability - and that's fine!")
			return nil
		})

	app.AddCommand(simpleCmd)

	return app.Run(args)
}
