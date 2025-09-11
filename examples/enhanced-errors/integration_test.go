// integration_test.go: enhanced-errors integration tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// TestIntegrationErrorProperties tests error properties without output capture
func TestIntegrationErrorProperties(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectError     bool
		expectExitCode  int
		expectRetryable bool
		expectSeverity  string
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			// Verify error expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify error properties
			if tt.expectError {
				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("expected OrpheusError but got %T", err)
					return
				}

				if orpheusErr.ExitCode() != tt.expectExitCode {
					t.Errorf("expected exit code %d but got %d", tt.expectExitCode, orpheusErr.ExitCode())
				}

				if orpheusErr.IsRetryable() != tt.expectRetryable {
					t.Errorf("expected retryable=%v but got %v", tt.expectRetryable, orpheusErr.IsRetryable())
				}

				// Verify user message is set
				if orpheusErr.UserMessage() == "" {
					t.Errorf("expected user message but got empty string")
				}
			}
		})
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
