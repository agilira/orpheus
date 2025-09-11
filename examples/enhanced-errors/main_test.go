// main_test.go: enhanced-errors example tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// TestValidateCommand tests the validate command behavior
func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		expectCode    string
		expectRetry   bool
		expectUserMsg string
	}{
		{
			name:          "missing data parameter",
			args:          []string{"validate"},
			expectError:   true,
			expectCode:    "ORF1000",
			expectRetry:   false,
			expectUserMsg: "Please provide data to validate using the --data flag",
		},
		{
			name:          "invalid data",
			args:          []string{"validate", "--data", "invalid"},
			expectError:   true,
			expectCode:    "ORF1000",
			expectRetry:   false,
			expectUserMsg: "The provided data does not match the expected format",
		},
		{
			name:        "valid data",
			args:        []string{"validate", "--data", "valid-content"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("expected OrpheusError but got %T", err)
					return
				}

				if string(orpheusErr.ErrorCode()) != tt.expectCode {
					t.Errorf("expected error code %s but got %s", tt.expectCode, orpheusErr.ErrorCode())
				}

				if orpheusErr.IsRetryable() != tt.expectRetry {
					t.Errorf("expected retryable=%v but got %v", tt.expectRetry, orpheusErr.IsRetryable())
				}

				if tt.expectUserMsg != "" && orpheusErr.UserMessage() != tt.expectUserMsg {
					t.Errorf("expected user message '%s' but got '%s'", tt.expectUserMsg, orpheusErr.UserMessage())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestConnectCommand tests the connect command behavior
func TestConnectCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		expectCode    string
		expectRetry   bool
		expectUserMsg string
	}{
		{
			name:          "timeout too short",
			args:          []string{"connect", "--timeout", "3"},
			expectError:   true,
			expectCode:    "ORF1001",
			expectRetry:   true,
			expectUserMsg: "Connection timeout is too short for reliable connection",
		},
		{
			name:          "connection failure",
			args:          []string{"connect", "--timeout", "8"},
			expectError:   true,
			expectCode:    "ORF1001",
			expectRetry:   true,
			expectUserMsg: "Unable to establish connection within the specified timeout",
		},
		{
			name:        "successful connection",
			args:        []string{"connect", "--timeout", "15"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("expected OrpheusError but got %T", err)
					return
				}

				if string(orpheusErr.ErrorCode()) != tt.expectCode {
					t.Errorf("expected error code %s but got %s", tt.expectCode, orpheusErr.ErrorCode())
				}

				if orpheusErr.IsRetryable() != tt.expectRetry {
					t.Errorf("expected retryable=%v but got %v", tt.expectRetry, orpheusErr.IsRetryable())
				}

				if tt.expectUserMsg != "" && orpheusErr.UserMessage() != tt.expectUserMsg {
					t.Errorf("expected user message '%s' but got '%s'", tt.expectUserMsg, orpheusErr.UserMessage())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestProcessCommand tests the process command behavior
func TestProcessCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		expectCode    string
		expectUserMsg string
	}{
		{
			name:          "missing file parameter",
			args:          []string{"process"},
			expectError:   true,
			expectCode:    "ORF1000",
			expectUserMsg: "Please specify a file to process using the --file flag",
		},
		{
			name:          "file not found",
			args:          []string{"process", "--file", "nonexistent.txt"},
			expectError:   true,
			expectCode:    "ORF1002",
			expectUserMsg: "The file you specified could not be found",
		},
		{
			name:        "file creation",
			args:        []string{"process", "--file", "nonexistent.txt", "--create"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("expected OrpheusError but got %T", err)
					return
				}

				if string(orpheusErr.ErrorCode()) != tt.expectCode {
					t.Errorf("expected error code %s but got %s", tt.expectCode, orpheusErr.ErrorCode())
				}

				if tt.expectUserMsg != "" && orpheusErr.UserMessage() != tt.expectUserMsg {
					t.Errorf("expected user message '%s' but got '%s'", tt.expectUserMsg, orpheusErr.UserMessage())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCriticalCommand tests the critical command behavior
func TestCriticalCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectError   bool
		expectCode    string
		expectUserMsg string
		expectExit    int
	}{
		{
			name:        "no simulation",
			args:        []string{"critical"},
			expectError: false,
		},
		{
			name:          "critical error simulation",
			args:          []string{"critical", "--simulate"},
			expectError:   true,
			expectCode:    "ORF1003",
			expectUserMsg: "A critical system error has occurred",
			expectExit:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}

				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("expected OrpheusError but got %T", err)
					return
				}

				if string(orpheusErr.ErrorCode()) != tt.expectCode {
					t.Errorf("expected error code %s but got %s", tt.expectCode, orpheusErr.ErrorCode())
				}

				if tt.expectUserMsg != "" && orpheusErr.UserMessage() != tt.expectUserMsg {
					t.Errorf("expected user message '%s' but got '%s'", tt.expectUserMsg, orpheusErr.UserMessage())
				}

				if tt.expectExit != 0 && orpheusErr.ExitCode() != tt.expectExit {
					t.Errorf("expected exit code %d but got %d", tt.expectExit, orpheusErr.ExitCode())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestErrorTypes tests error type classification
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		testFunc func(*orpheus.OrpheusError) bool
	}{
		{
			name:     "validation error type",
			args:     []string{"validate"},
			testFunc: func(err *orpheus.OrpheusError) bool { return err.IsValidationError() },
		},
		{
			name:     "execution error type",
			args:     []string{"connect", "--timeout", "3"},
			testFunc: func(err *orpheus.OrpheusError) bool { return err.IsExecutionError() },
		},
		{
			name:     "not found error type",
			args:     []string{"process", "--file", "nonexistent.txt"},
			testFunc: func(err *orpheus.OrpheusError) bool { return err.IsNotFoundError() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if err == nil {
				t.Errorf("expected error but got none")
				return
			}

			orpheusErr, ok := err.(*orpheus.OrpheusError)
			if !ok {
				t.Errorf("expected OrpheusError but got %T", err)
				return
			}

			if !tt.testFunc(orpheusErr) {
				t.Errorf("error type test failed for %s", tt.name)
			}
		})
	}
}

// TestHelpGeneration tests that help is generated correctly
func TestHelpGeneration(t *testing.T) {
	app := createTestApp()

	// Test main help
	err := app.Run([]string{"--help"})
	if err != nil {
		t.Errorf("help should not return error: %v", err)
	}

	// Test command help
	err = app.Run([]string{"validate", "--help"})
	if err != nil {
		t.Errorf("command help should not return error: %v", err)
	}
}

// createTestApp creates a test instance of the application
func createTestApp() *orpheus.App {
	app := orpheus.New("enhanced-errors").
		SetDescription("Demonstrate enhanced error handling with go-errors integration").
		SetVersion("1.0.0")

	validateCmd := orpheus.NewCommand("validate", "Validate input data").
		AddFlag("data", "d", "", "Data to validate (required)").
		AddFlag("format", "f", "json", "Expected format").
		SetHandler(handleValidate)

	connectCmd := orpheus.NewCommand("connect", "Connect to remote service").
		AddIntFlag("timeout", "t", 30, "Connection timeout in seconds").
		AddFlag("host", "h", "localhost", "Remote host").
		AddIntFlag("port", "p", 8080, "Remote port").
		SetHandler(handleConnect)

	processCmd := orpheus.NewCommand("process", "Process a file").
		AddFlag("file", "f", "", "File to process (required)").
		AddBoolFlag("create", "c", false, "Create file if missing").
		SetHandler(handleProcess)

	criticalCmd := orpheus.NewCommand("critical", "Simulate critical system error").
		AddBoolFlag("simulate", "s", false, "Simulate the error").
		SetHandler(handleCritical)

	app.AddCommand(validateCmd)
	app.AddCommand(connectCmd)
	app.AddCommand(processCmd)
	app.AddCommand(criticalCmd)

	return app
}
