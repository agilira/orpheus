// Enhanced Orpheus app: enhanced-errors example
//
// This example demonstrates Orpheus's enhanced error handling with go-errors integration,
// showing structured errors, user-friendly messages, context information, and retry semantics.
//
// Usage examples:
//   enhanced-errors validate --data invalid
//   enhanced-errors connect --timeout 1
//   enhanced-errors process --file nonexistent.txt
//   DEBUG=1 enhanced-errors validate --data invalid  (shows technical details)
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func main() {
	app := orpheus.New("enhanced-errors").
		SetDescription("Demonstrate enhanced error handling with go-errors integration").
		SetVersion("1.0.0")

	// Add validate command to show validation errors with context
	validateCmd := orpheus.NewCommand("validate", "Validate input data").
		AddFlag("data", "d", "", "Data to validate (required)").
		AddFlag("format", "f", "json", "Expected format").
		SetHandler(handleValidate)

	// Add connect command to show retryable execution errors
	connectCmd := orpheus.NewCommand("connect", "Connect to remote service").
		AddIntFlag("timeout", "t", 30, "Connection timeout in seconds").
		AddFlag("host", "h", "localhost", "Remote host").
		AddIntFlag("port", "p", 8080, "Remote port").
		SetHandler(handleConnect)

	// Add process command to show not found errors
	processCmd := orpheus.NewCommand("process", "Process a file").
		AddFlag("file", "f", "", "File to process (required)").
		AddBoolFlag("create", "c", false, "Create file if missing").
		SetHandler(handleProcess)

	// Add critical command to show critical internal errors
	criticalCmd := orpheus.NewCommand("critical", "Simulate critical system error").
		AddBoolFlag("simulate", "s", false, "Simulate the error").
		SetHandler(handleCritical)

	app.AddCommand(validateCmd)
	app.AddCommand(connectCmd)
	app.AddCommand(processCmd)
	app.AddCommand(criticalCmd)

	// Enhanced error handling with detailed information
	if err := app.Run(os.Args[1:]); err != nil {
		if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
			// Always show user-friendly message
			fmt.Fprintf(os.Stderr, "Error: %s\n", orpheusErr.UserMessage())

			// Show technical details in debug mode
			if os.Getenv("DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "\nTechnical Details:\n")
				fmt.Fprintf(os.Stderr, "  Error: %s\n", orpheusErr.Error())
				fmt.Fprintf(os.Stderr, "  Code:  %s\n", orpheusErr.ErrorCode())
			}

			// Show retry information
			if orpheusErr.IsRetryable() {
				fmt.Fprintf(os.Stderr, "\nThis operation can be retried\n")
			}

			// Show error type
			if orpheusErr.IsValidationError() {
				fmt.Fprintf(os.Stderr, "Tip: Check your input parameters\n")
			} else if orpheusErr.IsExecutionError() {
				fmt.Fprintf(os.Stderr, "Tip: Check system resources and connectivity\n")
			} else if orpheusErr.IsNotFoundError() {
				fmt.Fprintf(os.Stderr, "Tip: Verify the resource exists and you have access\n")
			}

			os.Exit(orpheusErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

func handleValidate(ctx *orpheus.Context) error {
	data := ctx.GetFlagString("data")
	format := ctx.GetFlagString("format")

	if data == "" {
		return orpheus.ValidationError("validate", "data parameter is required").
			WithUserMessage("Please provide data to validate using the --data flag").
			WithContext("expected_format", format).
			WithContext("usage", "enhanced-errors validate --data 'your-data'").
			WithContext("examples", []string{
				`enhanced-errors validate --data '{"name":"test"}'`,
				`enhanced-errors validate --data 'valid-string' --format text`,
			})
	}

	// Simulate validation logic
	if data == "invalid" {
		return orpheus.ValidationError("validate", "data contains invalid content").
			WithUserMessage("The provided data does not match the expected format").
			WithContext("provided_data", data).
			WithContext("expected_format", format).
			WithContext("validation_rules", []string{
				"Must not be 'invalid'",
				"Must be valid " + format,
				"Must not be empty",
			}).
			WithContext("valid_examples", []string{
				`{"valid": "json"}`,
				"valid-text-string",
			}).
			WithSeverity("warning")
	}

	fmt.Printf("Data validated successfully: %s (format: %s)\n", data, format)
	return nil
}

func handleConnect(ctx *orpheus.Context) error {
	timeout := ctx.GetFlagInt("timeout")
	host := ctx.GetFlagString("host")
	port := ctx.GetFlagInt("port")

	if timeout < 5 {
		return orpheus.ExecutionError("connect", "connection timeout too short").
			WithUserMessage("Connection timeout is too short for reliable connection").
			WithContext("provided_timeout", timeout).
			WithContext("minimum_timeout", 5).
			WithContext("recommended_timeout", 30).
			WithContext("host", host).
			WithContext("port", port).
			AsRetryable().
			WithSeverity("warning")
	}

	// Simulate connection attempt
	fmt.Printf("Connecting to %s:%d (timeout: %ds)...\n", host, port, timeout)
	time.Sleep(100 * time.Millisecond) // Simulate connection time

	// Simulate connection failure for demonstration
	if timeout < 10 {
		return orpheus.ExecutionError("connect", "connection failed: timeout exceeded").
			WithUserMessage("Unable to establish connection within the specified timeout").
			WithContext("host", host).
			WithContext("port", port).
			WithContext("timeout_used", timeout).
			WithContext("connection_attempt", 1).
			WithContext("last_error", "dial tcp: i/o timeout").
			WithContext("suggested_timeout", 30).
			AsRetryable().
			WithSeverity("error")
	}

	fmt.Printf("Connected successfully to %s:%d\n", host, port)
	return nil
}

func handleProcess(ctx *orpheus.Context) error {
	file := ctx.GetFlagString("file")
	create := ctx.GetFlagBool("create")

	if file == "" {
		return orpheus.ValidationError("process", "file parameter is required").
			WithUserMessage("Please specify a file to process using the --file flag").
			WithContext("usage", "enhanced-errors process --file path/to/file.txt").
			WithContext("supported_formats", []string{".txt", ".json", ".yaml", ".xml"})
	}

	// Simulate file check
	if file == "nonexistent.txt" && !create {
		return orpheus.NotFoundError("process", "specified file does not exist").
			WithUserMessage("The file you specified could not be found").
			WithContext("requested_file", file).
			WithContext("current_directory", getCurrentDir()).
			WithContext("create_flag", create).
			WithContext("suggestion", "use --create flag to create the file").
			WithContext("alternative", "verify the file path is correct").
			WithSeverity("warning")
	}

	if create && file == "nonexistent.txt" {
		fmt.Printf("Creating file: %s\n", file)
		// Simulate file creation
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("Processing file: %s\n", file)
	return nil
}

func handleCritical(ctx *orpheus.Context) error {
	simulate := ctx.GetFlagBool("simulate")

	if !simulate {
		fmt.Println("Use --simulate flag to see a critical error example")
		return nil
	}

	// Simulate a critical system error
	return orpheus.InternalError("system resource exhaustion detected").
		WithUserMessage("A critical system error has occurred").
		WithContext("error_type", "resource_exhaustion").
		WithContext("affected_resources", []string{"memory", "file_descriptors"}).
		WithContext("system_load", "95%").
		WithContext("available_memory", "< 100MB").
		WithContext("recommended_action", "restart application or increase resources").
		WithContext("contact_support", true).
		WithSeverity("critical")
}

func getCurrentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "unknown"
}
