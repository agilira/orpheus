// integration_test.go: Orpheus app: enhanced-errors integration tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// createTestApp creates a test app with the same configuration as main
func createTestApp() *orpheus.App {
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

	return app
}

// TestIntegrationErrorProperties tests error properties without output capture
func TestIntegrationErrorProperties(t *testing.T) {
	tests := createIntegrationTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)
			validateIntegrationResult(t, err, tt)
		})
	}
}

// integrationTest represents a test case for integration testing
type integrationTest struct {
	name            string
	args            []string
	expectError     bool
	expectExitCode  int
	expectRetryable bool
}

// createIntegrationTests creates test cases for integration testing
func createIntegrationTests() []integrationTest {
	return []integrationTest{
		{
			name:            "validation error properties",
			args:            []string{"validate"},
			expectError:     true,
			expectExitCode:  1,
			expectRetryable: false,
		},
		{
			name:            "retryable execution error",
			args:            []string{"connect", "--timeout", "3"},
			expectError:     true,
			expectExitCode:  1,
			expectRetryable: true,
		},
		{
			name:            "critical error with exit code 2",
			args:            []string{"critical", "--simulate"},
			expectError:     true,
			expectExitCode:  2,
			expectRetryable: false,
		},
	}
}

// validateIntegrationResult validates the result of integration test execution
func validateIntegrationResult(t *testing.T, err error, tt integrationTest) {
	if err := validateErrorExpectation(t, err, tt.expectError); err != nil {
		return
	}

	if tt.expectError {
		validateIntegrationErrorProperties(t, err, tt)
	}
}

// validateErrorExpectation validates whether error matches expectation
func validateErrorExpectation(t *testing.T, err error, expectError bool) error {
	if expectError && err == nil {
		t.Errorf("expected error but got none")
		return fmt.Errorf("missing expected error")
	}
	if !expectError && err != nil {
		t.Errorf("unexpected error: %v", err)
		return fmt.Errorf("unexpected error")
	}
	return nil
}

// validateIntegrationErrorProperties validates specific error properties for integration tests
func validateIntegrationErrorProperties(t *testing.T, err error, tt integrationTest) {
	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Errorf("expected OrpheusError but got %T", err)
		return
	}

	validateExitCode(t, orpheusErr, tt.expectExitCode)
	validateRetryable(t, orpheusErr, tt.expectRetryable)
	validateUserMessage(t, orpheusErr)
}

// validateExitCode validates the exit code property
func validateExitCode(t *testing.T, orpheusErr *orpheus.OrpheusError, expected int) {
	if orpheusErr.ExitCode() != expected {
		t.Errorf("expected exit code %d but got %d", expected, orpheusErr.ExitCode())
	}
}

// validateRetryable validates the retryable property
func validateRetryable(t *testing.T, orpheusErr *orpheus.OrpheusError, expected bool) {
	if orpheusErr.IsRetryable() != expected {
		t.Errorf("expected retryable=%v but got %v", expected, orpheusErr.IsRetryable())
	}
}

// validateUserMessage validates that user message is set
func validateUserMessage(t *testing.T, orpheusErr *orpheus.OrpheusError) {
	if orpheusErr.UserMessage() == "" {
		t.Errorf("expected user message but got empty string")
	}
}

// TestErrorContextAccumulation tests that context is properly accumulated
func TestErrorContextAccumulation(t *testing.T) {
	app := createTestApp()
	err := app.Run([]string{"validate", "--data", "invalid", "--format", "xml"})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Fatalf("expected OrpheusError but got %T", err)
	}

	// Verify the error contains expected context information
	errorStr := orpheusErr.Error()

	// Should contain command context
	if !strings.Contains(errorStr, "validate") {
		t.Errorf("expected error to contain command context 'validate'")
	}

	// Should be a validation error
	if !orpheusErr.IsValidationError() {
		t.Errorf("expected validation error")
	}

	// Should have proper error code
	if string(orpheusErr.ErrorCode()) != "ORF1000" {
		t.Errorf("expected error code ORF1000 but got %s", orpheusErr.ErrorCode())
	}

	// Should have user message
	userMsg := orpheusErr.UserMessage()
	if userMsg == "" {
		t.Errorf("expected user message but got empty string")
	}
}

// TestErrorChaining tests that errors can be properly unwrapped
func TestErrorChaining(t *testing.T) {
	app := createTestApp()
	err := app.Run([]string{"critical", "--simulate"})

	if err == nil {
		t.Fatal("expected error but got none")
	}

	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Fatalf("expected OrpheusError but got %T", err)
	}

	// Test error unwrapping
	underlyingErr := orpheusErr.Unwrap()
	if underlyingErr == nil {
		t.Errorf("expected underlying error but got nil")
	}
}

// TestConcurrentErrorCreation tests thread safety of error creation
func TestConcurrentErrorCreation(t *testing.T) {
	const numGoroutines = 100
	const numIterations = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			app := createTestApp()

			for j := 0; j < numIterations; j++ {
				// Test different error types concurrently
				switch j % 4 {
				case 0:
					app.Run([]string{"validate"})
				case 1:
					app.Run([]string{"connect", "--timeout", "3"})
				case 2:
					app.Run([]string{"process", "--file", "nonexistent.txt"})
				case 3:
					app.Run([]string{"critical", "--simulate"})
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestErrorMessageConsistency tests that error messages are consistent
func TestErrorMessageConsistency(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedPrefix string
	}{
		{
			name:           "validation prefix",
			args:           []string{"validate"},
			expectedPrefix: "command 'validate':",
		},
		{
			name:           "connect prefix",
			args:           []string{"connect", "--timeout", "3"},
			expectedPrefix: "command 'connect':",
		},
		{
			name:           "process prefix",
			args:           []string{"process", "--file", "nonexistent.txt"},
			expectedPrefix: "command 'process':",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if err == nil {
				t.Fatal("expected error but got none")
			}

			orpheusErr, ok := err.(*orpheus.OrpheusError)
			if !ok {
				t.Fatalf("expected OrpheusError but got %T", err)
			}

			errorStr := orpheusErr.Error()
			if !strings.HasPrefix(errorStr, tt.expectedPrefix) {
				t.Errorf("expected error to start with '%s' but got: %s", tt.expectedPrefix, errorStr)
			}
		})
	}
}
