// errors_test.go: errors tests for Orpheus with go-errors integration
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestOrpheusErrorCreation(t *testing.T) {
	err := orpheus.NewOrpheusError(orpheus.ErrCodeValidation, "test", "test message")

	if err.ErrorCode() != orpheus.ErrCodeValidation {
		t.Errorf("expected validation error code, got %s", err.ErrorCode())
	}

	if err.Command != "test" {
		t.Errorf("expected command 'test', got '%s'", err.Command)
	}

	if !err.IsValidationError() {
		t.Error("expected validation error")
	}
}

func TestOrpheusErrorInterface(t *testing.T) {
	err := orpheus.ValidationError("test", "validation failed")

	// Test error interface
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("error message should not be empty")
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

	if err.ErrorCode() != orpheus.ErrCodeValidation {
		t.Errorf("expected validation code, got %s", err.ErrorCode())
	}

	// Test user message
	userMsg := err.UserMessage()
	if userMsg != "Invalid input or missing required arguments" {
		t.Errorf("unexpected user message: %s", userMsg)
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

	if err.ErrorCode() != orpheus.ErrCodeExecution {
		t.Errorf("expected execution code, got %s", err.ErrorCode())
	}

	// Test user message
	userMsg := err.UserMessage()
	if userMsg != "Command execution failed" {
		t.Errorf("unexpected user message: %s", userMsg)
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

	if err.ErrorCode() != orpheus.ErrCodeNotFound {
		t.Errorf("expected not found code, got %s", err.ErrorCode())
	}

	// Test user message
	userMsg := err.UserMessage()
	if userMsg != "Command or resource not found" {
		t.Errorf("unexpected user message: %s", userMsg)
	}
}

func TestInternalError(t *testing.T) {
	err := orpheus.InternalError("internal failure")

	if err.ErrorCode() != orpheus.ErrCodeInternal {
		t.Errorf("expected internal code, got %s", err.ErrorCode())
	}

	if err.Command != "" {
		t.Errorf("expected empty command for internal error, got '%s'", err.Command)
	}

	if err.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for internal error, got %d", err.ExitCode())
	}

	// Test user message
	userMsg := err.UserMessage()
	if userMsg != "An internal error occurred" {
		t.Errorf("unexpected user message: %s", userMsg)
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		errorCode string
		factory   func() *orpheus.OrpheusError
	}{
		{"validation", "ORF1000", func() *orpheus.OrpheusError { return orpheus.ValidationError("test", "msg") }},
		{"execution", "ORF1001", func() *orpheus.OrpheusError { return orpheus.ExecutionError("test", "msg") }},
		{"not_found", "ORF1002", func() *orpheus.OrpheusError { return orpheus.NotFoundError("test", "msg") }},
		{"internal", "ORF1003", func() *orpheus.OrpheusError { return orpheus.InternalError("msg") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.factory()
			if string(err.ErrorCode()) != tt.errorCode {
				t.Errorf("expected error code '%s', got '%s'", tt.errorCode, string(err.ErrorCode()))
			}
		})
	}
}

func TestErrorWithEmptyCommand(t *testing.T) {
	err := orpheus.ValidationError("", "no command specified")

	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("error message should not be empty")
	}
}

func TestEnhancedErrorFeatures(t *testing.T) {
	err := orpheus.ValidationError("test", "validation failed").
		WithUserMessage("Please check your input").
		WithContext("field", "username").
		WithContext("value", "invalid@user").
		AsRetryable().
		WithSeverity("warning")

	// Test fluent API worked
	if err.UserMessage() != "Please check your input" {
		t.Errorf("unexpected user message: %s", err.UserMessage())
	}

	if !err.IsRetryable() {
		t.Error("error should be retryable")
	}

	// Test error is still the same type
	if !err.IsValidationError() {
		t.Error("error should still be validation error")
	}
}

func TestErrorUnwrap(t *testing.T) {
	err := orpheus.ValidationError("test", "validation failed")

	// Test unwrap returns the underlying go-errors
	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Error("unwrap should return underlying error")
	}

	// Test error chain compatibility
	if !errors.Is(err, unwrapped) {
		t.Error("error chain should work with errors.Is")
	}
}

func TestErrorJSONSerialization(t *testing.T) {
	err := orpheus.ValidationError("test", "validation failed").
		WithUserMessage("User friendly message").
		WithContext("field", "test")

	// Test that the underlying go-errors can be JSON marshaled
	// (we can't directly marshal OrpheusError, but we can access the underlying error)
	underlying := err.Unwrap()

	jsonData, jsonErr := json.Marshal(underlying)
	if jsonErr != nil {
		t.Errorf("failed to marshal error to JSON: %v", jsonErr)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}
}
