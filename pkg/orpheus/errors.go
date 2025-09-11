// errors.go: errors definitions for Orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
)

// ErrorType represents the type of error that occurred.
type ErrorType string

const (
	// ErrorValidation indicates a validation error (bad input, missing args, etc.)
	ErrorValidation ErrorType = "validation"

	// ErrorExecution indicates an error during command execution
	ErrorExecution ErrorType = "execution"

	// ErrorNotFound indicates a command or resource was not found
	ErrorNotFound ErrorType = "not_found"

	// ErrorInternal indicates an internal framework error
	ErrorInternal ErrorType = "internal"
)

// OrpheusError represents an error that occurred within the Orpheus framework.
type OrpheusError struct {
	Type    ErrorType
	Command string
	Message string
	Code    int
}

// NewOrpheusError creates a new OrpheusError.
func NewOrpheusError(errorType ErrorType, command, message string, code int) *OrpheusError {
	return &OrpheusError{
		Type:    errorType,
		Command: command,
		Message: message,
		Code:    code,
	}
}

// Error implements the error interface.
func (e *OrpheusError) Error() string {
	if e.Command != "" {
		return fmt.Sprintf("%s error in command '%s': %s", e.Type, e.Command, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// ExitCode returns the suggested exit code for this error.
func (e *OrpheusError) ExitCode() int {
	return e.Code
}

// IsValidationError returns true if this is a validation error.
func (e *OrpheusError) IsValidationError() bool {
	return e.Type == ErrorValidation
}

// IsExecutionError returns true if this is an execution error.
func (e *OrpheusError) IsExecutionError() bool {
	return e.Type == ErrorExecution
}

// IsNotFoundError returns true if this is a not found error.
func (e *OrpheusError) IsNotFoundError() bool {
	return e.Type == ErrorNotFound
}

// ValidationError creates a validation error.
func ValidationError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrorValidation, command, message, 1)
}

// ExecutionError creates an execution error.
func ExecutionError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrorExecution, command, message, 1)
}

// NotFoundError creates a not found error.
func NotFoundError(command, message string) *OrpheusError {
	return NewOrpheusError(ErrorNotFound, command, message, 1)
}

// InternalError creates an internal error.
func InternalError(message string) *OrpheusError {
	return NewOrpheusError(ErrorInternal, "", message, 2)
}
