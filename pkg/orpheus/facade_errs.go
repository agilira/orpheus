// facade_errs.go: the errs API under its v1.11.0 names
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	goerrors "github.com/agilira/go-errors"

	"github.com/agilira/orpheus/errs"
)

// Error is errs.Error.
type Error = errs.Error

// OrpheusError is an alias for Error for backward compatibility
type OrpheusError = errs.Error

// Error codes, see package errs.
const (
	ErrCodeValidation         = errs.ErrCodeValidation
	ErrCodeExecution          = errs.ErrCodeExecution
	ErrCodeNotFound           = errs.ErrCodeNotFound
	ErrCodeInternal           = errs.ErrCodeInternal
	ErrCodeStorageValidation  = errs.ErrCodeStorageValidation
	ErrCodeStorageExecution   = errs.ErrCodeStorageExecution
	ErrCodeStorageNotFound    = errs.ErrCodeStorageNotFound
	ErrCodeStorageUnavailable = errs.ErrCodeStorageUnavailable
	ErrCodePluginError        = errs.ErrCodePluginError
)

// NewError calls errs.NewError.
func NewError(code goerrors.ErrorCode, command, message string) *Error {
	return errs.NewError(code, command, message)
}

// NewOrpheusError is an alias for NewError for backward compatibility
func NewOrpheusError(code goerrors.ErrorCode, command, message string) *Error {
	return errs.NewError(code, command, message)
}

// ValidationError calls errs.ValidationError.
func ValidationError(command, message string) *Error { return errs.ValidationError(command, message) }

// ExecutionError calls errs.ExecutionError.
func ExecutionError(command, message string) *Error { return errs.ExecutionError(command, message) }

// NotFoundError calls errs.NotFoundError.
func NotFoundError(command, message string) *Error { return errs.NotFoundError(command, message) }

// InternalError calls errs.InternalError.
func InternalError(message string) *Error { return errs.InternalError(message) }
