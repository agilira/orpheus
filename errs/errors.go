// Package errs defines the structured error type of Orpheus and its error
// codes, built on github.com/agilira/go-errors.
//
// Every error a command, the framework or the storage layer returns is an
// *Error carrying a code (ErrCodeValidation, ErrCodeStorageNotFound, ...),
// a suggested exit code, a user message and context.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package errs

import (
	"fmt"

	goerrors "github.com/agilira/go-errors"
)

// Orpheus-specific error codes using go-errors framework
const (
	// ErrCodeValidation indicates a validation error (bad input, missing args, etc.)
	ErrCodeValidation goerrors.ErrorCode = "ORF1000"

	// ErrCodeExecution indicates an error during command execution
	ErrCodeExecution goerrors.ErrorCode = "ORF1001"

	// ErrCodeNotFound indicates a command or resource was not found
	ErrCodeNotFound goerrors.ErrorCode = "ORF1002"

	// ErrCodeInternal indicates an internal framework error
	ErrCodeInternal goerrors.ErrorCode = "ORF1003"
)

// Storage-specific error codes
const (
	// ErrCodeStorageValidation indicates invalid storage configuration or parameters
	ErrCodeStorageValidation goerrors.ErrorCode = "ORF2000"

	// ErrCodeStorageExecution indicates a storage operation failure
	ErrCodeStorageExecution goerrors.ErrorCode = "ORF2001"

	// ErrCodeStorageNotFound indicates a requested key was not found
	ErrCodeStorageNotFound goerrors.ErrorCode = "ORF2002"

	// ErrCodeStorageUnavailable indicates the storage backend is not available
	ErrCodeStorageUnavailable goerrors.ErrorCode = "ORF2003"

	// ErrCodePluginError indicates a plugin loading or execution error
	ErrCodePluginError goerrors.ErrorCode = "ORF2004"
)

// Error represents an enhanced error with go-errors capabilities
// This follows go-errors patterns while avoiding stuttering (orpheus.Error vs orpheus.OrpheusError)
type Error struct {
	goError *goerrors.Error
	Command string
}

// NewError creates a new enhanced Error using go-errors framework
func NewError(code goerrors.ErrorCode, command, message string) *Error {
	err := goerrors.New(code, message).
		WithContext("command", command).
		WithSeverity("error")

	return &Error{
		goError: err,
		Command: command,
	}
}

// NewErrorWithCause is NewError with cause kept in the error chain, so that
// errors.Is and errors.As on the result reach it.
func NewErrorWithCause(code goerrors.ErrorCode, command, message string, cause error) *Error {
	e := NewError(code, command, message)
	e.goError.Cause = cause
	return e
}

// Error implements the error interface with enhanced formatting
func (e *Error) Error() string {
	if e.Command != "" {
		return fmt.Sprintf("command '%s': %s", e.Command, e.goError.Error())
	}
	return e.goError.Error()
}

// ErrorCode returns the error code from the underlying go-errors
func (e *Error) ErrorCode() goerrors.ErrorCode {
	return e.goError.ErrorCode()
}

// ExitCode returns the suggested exit code based on error type
func (e *Error) ExitCode() int {
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
func (e *Error) IsValidationError() bool {
	return e.ErrorCode() == ErrCodeValidation
}

// IsExecutionError returns true if this is an execution error
func (e *Error) IsExecutionError() bool {
	return e.ErrorCode() == ErrCodeExecution
}

// IsNotFoundError returns true if this is a not found error
func (e *Error) IsNotFoundError() bool {
	return e.ErrorCode() == ErrCodeNotFound
}

// UserMessage returns the user-friendly message
func (e *Error) UserMessage() string {
	return e.goError.UserMessage()
}

// IsRetryable returns whether the error is retryable
func (e *Error) IsRetryable() bool {
	return e.goError.IsRetryable()
}

// WithUserMessage adds a user-friendly message and returns the error for chaining
func (e *Error) WithUserMessage(msg string) *Error {
	e.goError = e.goError.WithUserMessage(msg)
	return e
}

// WithContext adds context information and returns the error for chaining
func (e *Error) WithContext(key string, value any) *Error {
	e.goError = e.goError.WithContext(key, value)
	return e
}

// AsRetryable marks the error as retryable and returns the error for chaining
func (e *Error) AsRetryable() *Error {
	e.goError = e.goError.AsRetryable()
	return e
}

// WithSeverity sets the severity level and returns the error for chaining
func (e *Error) WithSeverity(severity string) *Error {
	e.goError = e.goError.WithSeverity(severity)
	return e
}

// Unwrap returns the underlying go-errors Error for error chain compatibility
func (e *Error) Unwrap() error {
	return e.goError
}

// ValidationError creates a validation error with enhanced go-errors capabilities
func ValidationError(command, message string) *Error {
	return NewError(ErrCodeValidation, command, message).
		WithUserMessage("Invalid input or missing required arguments").
		WithSeverity("warning")
}

// ExecutionError creates an execution error with enhanced go-errors capabilities
func ExecutionError(command, message string) *Error {
	return NewError(ErrCodeExecution, command, message).
		WithUserMessage("Command execution failed").
		WithSeverity("error")
}

// NotFoundError creates a not found error with enhanced go-errors capabilities
func NotFoundError(command, message string) *Error {
	return NewError(ErrCodeNotFound, command, message).
		WithUserMessage("Command or resource not found").
		WithSeverity("warning")
}

// InternalError creates an internal error with enhanced go-errors capabilities
func InternalError(message string) *Error {
	return NewError(ErrCodeInternal, "", message).
		WithUserMessage("An internal error occurred").
		WithSeverity("critical")
}
