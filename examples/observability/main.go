// main.go: Orpheus app: observability example
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// SimpleLogger demonstrates a basic logger implementation
type SimpleLogger struct {
	prefix string
}

func NewSimpleLogger(prefix string) *SimpleLogger {
	return &SimpleLogger{prefix: prefix}
}

func (l *SimpleLogger) Trace(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.log("TRACE", msg, fields...)
}

func (l *SimpleLogger) Debug(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.log("DEBUG", msg, fields...)
}

func (l *SimpleLogger) Info(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.log("INFO", msg, fields...)
}

func (l *SimpleLogger) Warn(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.log("WARN", msg, fields...)
}

func (l *SimpleLogger) Error(ctx context.Context, msg string, fields ...orpheus.Field) {
	l.log("ERROR", msg, fields...)
}

func (l *SimpleLogger) WithFields(fields ...orpheus.Field) orpheus.Logger {
	// For simplicity, return the same logger
	// In a real implementation, you'd create a new logger with additional fields
	return l
}

func (l *SimpleLogger) log(level, msg string, fields ...orpheus.Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s %s: %s", timestamp, l.prefix, level, msg)

	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

// SimpleAuditLogger demonstrates a basic audit logger implementation
type SimpleAuditLogger struct{}

func NewSimpleAuditLogger() *SimpleAuditLogger {
	return &SimpleAuditLogger{}
}

func (a *SimpleAuditLogger) LogCommand(ctx context.Context, command string, args []string, user string, fields ...orpheus.Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] AUDIT: user=%s command=%s args=%v\n", timestamp, user, command, args)
}

func (a *SimpleAuditLogger) LogAccess(ctx context.Context, resource string, action string, allowed bool, fields ...orpheus.Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	status := "ALLOWED"
	if !allowed {
		status = "DENIED"
	}
	fmt.Printf("[%s] ACCESS: resource=%s action=%s status=%s\n", timestamp, resource, action, status)
}

func (a *SimpleAuditLogger) LogSecurity(ctx context.Context, event string, severity string, fields ...orpheus.Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] SECURITY: event=%s severity=%s\n", timestamp, event, severity)
}

func (a *SimpleAuditLogger) LogPerformance(ctx context.Context, operation string, duration int64, fields ...orpheus.Field) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] PERFORMANCE: operation=%s duration=%dms\n", timestamp, operation, duration)
}

func main() {
	// Create logger and audit logger
	logger := NewSimpleLogger("MyApp")
	auditLogger := NewSimpleAuditLogger()

	// Create application with observability
	app := orpheus.New("observability-demo").
		SetDescription("Demonstration of Orpheus observability features").
		SetVersion("1.0.0").
		SetLogger(logger).
		SetAuditLogger(auditLogger)

	// Create a command that demonstrates observability usage
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

			// Simulate processing time
			time.Sleep(100 * time.Millisecond)

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

	// Add a command without observability to show it still works
	app.Command("simple", "Simple command without observability", func(ctx *orpheus.Context) error {
		fmt.Println("This command doesn't use observability - and that's fine!")
		return nil
	})

	// Run the application
	if err := app.Run(os.Args[1:]); err != nil {
		if logger := app.Logger(); logger != nil {
			logger.Error(context.Background(), "Application error",
				orpheus.ErrorField(err),
			)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
