// errors_test.go: errors tests for Orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestOrpheusErrorCreation(t *testing.T) {
	err := orpheus.NewOrpheusError(orpheus.ErrorValidation, "test", "test message", 1)

	if err.Type != orpheus.ErrorValidation {
		t.Errorf("expected validation error type, got %s", err.Type)
	}

	if err.Command != "test" {
		t.Errorf("expected command 'test', got '%s'", err.Command)
	}

	if err.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", err.Message)
	}

	if err.Code != 1 {
		t.Errorf("expected code 1, got %d", err.Code)
	}
}

func TestOrpheusErrorInterface(t *testing.T) {
	err := orpheus.ValidationError("test", "validation failed")

	// Test error interface
	errorMsg := err.Error()
	expected := "validation error in command 'test': validation failed"
	if errorMsg != expected {
		t.Errorf("expected '%s', got '%s'", expected, errorMsg)
	}

	// Test exit code
	if err.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", err.ExitCode())
	}
}

func TestValidationError(t *testing.T) {
	err := orpheus.ValidationError("test", "invalid input")

	if !err.IsValidationError() {
		t.Error("expected validation error")
	}

	if err.IsExecutionError() {
		t.Error("should not be execution error")
	}

	if err.IsNotFoundError() {
		t.Error("should not be not found error")
	}

	if err.Type != orpheus.ErrorValidation {
		t.Errorf("expected validation type, got %s", err.Type)
	}
}

func TestExecutionError(t *testing.T) {
	err := orpheus.ExecutionError("test", "execution failed")

	if !err.IsExecutionError() {
		t.Error("expected execution error")
	}

	if err.IsValidationError() {
		t.Error("should not be validation error")
	}

	if err.IsNotFoundError() {
		t.Error("should not be not found error")
	}

	if err.Type != orpheus.ErrorExecution {
		t.Errorf("expected execution type, got %s", err.Type)
	}
}

func TestNotFoundError(t *testing.T) {
	err := orpheus.NotFoundError("test", "command not found")

	if !err.IsNotFoundError() {
		t.Error("expected not found error")
	}

	if err.IsValidationError() {
		t.Error("should not be validation error")
	}

	if err.IsExecutionError() {
		t.Error("should not be execution error")
	}

	if err.Type != orpheus.ErrorNotFound {
		t.Errorf("expected not found type, got %s", err.Type)
	}
}

func TestInternalError(t *testing.T) {
	err := orpheus.InternalError("internal failure")

	if err.Type != orpheus.ErrorInternal {
		t.Errorf("expected internal type, got %s", err.Type)
	}

	if err.Command != "" {
		t.Errorf("expected empty command for internal error, got '%s'", err.Command)
	}

	if err.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for internal error, got %d", err.ExitCode())
	}

	// Test error message without command
	errorMsg := err.Error()
	expected := "internal error: internal failure"
	if errorMsg != expected {
		t.Errorf("expected '%s', got '%s'", expected, errorMsg)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorType orpheus.ErrorType
		expected  string
	}{
		{"validation", orpheus.ErrorValidation, "validation"},
		{"execution", orpheus.ErrorExecution, "execution"},
		{"not_found", orpheus.ErrorNotFound, "not_found"},
		{"internal", orpheus.ErrorInternal, "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.errorType) != tt.expected {
				t.Errorf("expected error type '%s', got '%s'", tt.expected, string(tt.errorType))
			}
		})
	}
}

func TestErrorWithEmptyCommand(t *testing.T) {
	err := orpheus.ValidationError("", "no command specified")

	errorMsg := err.Error()
	expected := "validation error: no command specified"
	if errorMsg != expected {
		t.Errorf("expected '%s', got '%s'", expected, errorMsg)
	}
}
