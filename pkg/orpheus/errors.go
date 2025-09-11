// errors.go: errors definitions for Orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"

	"github.com/agilira/go-errors"
)

// Orpheus-specific error codes using go-errors framework
const (
	// ErrCodeValidation indicates a validation error (bad input, missing args, etc.)
	ErrCodeValidation errors.ErrorCode = "ORF1000"

	// ErrCodeExecution indicates an error during command execution
	ErrCodeExecution errors.ErrorCode = "ORF1001"

	// ErrCodeNotFound indicates a command or resource was not found
	ErrCodeNotFound errors.ErrorCode = "ORF1002"

	// ErrCodeInternal indicates an internal framework error
	ErrCodeInternal errors.ErrorCode = "ORF1003"
)

// OrpheusError represents an enhanced error with go-errors capabilities
type OrpheusError struct {
	goError *errors.Error
	Command string
}

// NewOrpheusError creates a new enhanced OrpheusError using go-errors
func NewOrpheusError(code errors.ErrorCode, command, message string) *OrpheusError {
	err := errors.New(code, message).
		WithContext("command", command).
		WithSeverity("error")

	return &OrpheusError{
		goError: err,
		Command: command,
	}
}

// Error implements the error interface with enhanced formatting
func (e *OrpheusError) Error() string {
	if e.Command != "" {
		return fmt.Sprintf("command '%s': %s", e.Command, e.goError.Error())
	}
	return e.goError.Error()
}

// ErrorCode returns the error code from the underlying go-errors
func (e *OrpheusError) ErrorCode() errors.ErrorCode {
	return e.goError.ErrorCode()
}

// ExitCode returns the suggested exit code based on error type
func (e *OrpheusError) ExitCode() int {
	switch e.ErrorCode() {
	case ErrCodeValidation:
		return 1
	case ErrCodeExecution:
		return 1
	case ErrCodeNotFound:
		return 1
	case ErrCodeInternal:
		return 2
	default:
		return 1
	}
}

// IsValidationError returns true if this is a validation error
func (e *OrpheusError) IsValidationError() bool {
	return e.ErrorCode() == ErrCodeValidation
}

// IsExecutionError returns true if this is an execution error
func (e *OrpheusError) IsExecutionError() bool {
	return e.ErrorCode() == ErrCodeExecution
}

// IsNotFoundError returns true if this is a not found error
func (e *OrpheusError) IsNotFoundError() bool {
	return e.ErrorCode() == ErrCodeNotFound
}

// UserMessage returns the user-friendly message
func (e *OrpheusError) UserMessage() string {
	return e.goError.UserMessage()
}

// IsRetryable returns whether the error is retryable
func (e *OrpheusError) IsRetryable() bool {
	return e.goError.IsRetryable()
}

// WithUserMessage adds a user-friendly message and returns the error for chaining
func (e *OrpheusError) WithUserMessage(msg string) *OrpheusError {
	e.goError.WithUserMessage(msg)
	return e
}

// WithContext adds context information and returns the error for chaining
func (e *OrpheusError) WithContext(key string, value interface{}) *OrpheusError {
	e.goError.WithContext(key, value)
	return e
}

// AsRetryable marks the error as retryable and returns the error for chaining
func (e *OrpheusError) AsRetryable() *OrpheusError {
	e.goError.AsRetryable()
	return e
}

// WithSeverity sets the severity level and returns the error for chaining
func (e *OrpheusError) WithSeverity(severity string) *OrpheusError {
	e.goError.WithSeverity(severity)
	return e
}

// Unwrap returns the underlying go-errors Error for error chain compatibility
func (e *OrpheusError) Unwrap() error {
	return e.goError
}

// ValidationError creates a validation error with enhanced capabilities
func ValidationError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrCodeValidation, command, message).
		WithUserMessage("Invalid input or missing required arguments").
		WithSeverity("warning")
}

// ExecutionError creates an execution error with enhanced capabilities
func ExecutionError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrCodeExecution, command, message).
		WithUserMessage("Command execution failed").
		WithSeverity("error")
}

// NotFoundError creates a not found error with enhanced capabilities
func NotFoundError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrCodeNotFound, command, message).
		WithUserMessage("Command or resource not found").
		WithSeverity("warning")
}

// InternalError creates an internal error with enhanced capabilities
func InternalError(message string) *OrpheusError {
	return NewOrpheusError(ErrCodeInternal, "", message).
		WithUserMessage("An internal error occurred").
		WithSeverity("critical")
}
