// Storage Error System for Orpheus
//
// Provides structured error handling for storage operations with full context,
// following AGILira error handling standards. All storage errors include
// operation context, key information, and provider details for debugging.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"errors"
	"fmt"

	goerrors "github.com/agilira/go-errors"
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

// Common storage errors - these are sentinel errors for easy comparison
var (
	// ErrKeyNotFound is returned when a requested key doesn't exist
	ErrKeyNotFound = errors.New("key not found")

	// ErrKeyEmpty is returned when an empty key is provided
	ErrKeyEmpty = errors.New("key cannot be empty")

	// ErrKeyTooLong is returned when a key exceeds maximum length
	ErrKeyTooLong = errors.New("key exceeds maximum length")

	// ErrKeyInvalid is returned when a key contains invalid characters
	ErrKeyInvalid = errors.New("key contains invalid characters")

	// ErrValueTooLarge is returned when a value exceeds maximum size
	ErrValueTooLarge = errors.New("value exceeds maximum size")

	// ErrStorageNotConfigured is returned when storage is accessed but not configured
	ErrStorageNotConfigured = errors.New("storage not configured")

	// ErrStorageUnavailable is returned when storage backend is not available
	ErrStorageUnavailable = errors.New("storage backend unavailable")

	// ErrStorageClosed is returned when operations are attempted on closed storage
	ErrStorageClosed = errors.New("storage has been closed")

	// ErrPluginNotFound is returned when a storage plugin cannot be located
	ErrPluginNotFound = errors.New("storage plugin not found")

	// ErrPluginInvalid is returned when a plugin doesn't implement required interfaces
	ErrPluginInvalid = errors.New("storage plugin invalid")

	// ErrConfigInvalid is returned when storage configuration is invalid
	ErrConfigInvalid = errors.New("storage configuration invalid")
)

// Helper functions to create storage-specific Orpheus errors with proper context

// StorageGetError creates an error for failed Get operations
func StorageGetError(key string, err error) *Error {
	return NewError(ErrCodeStorageExecution, "storage", fmt.Sprintf("get operation failed for key '%s': %v", key, err)).
		WithContext("operation", "storage.Get").
		WithContext("key", key).
		WithSeverity("warning")
}

// StorageSetError creates an error for failed Set operations
func StorageSetError(key string, err error) *Error {
	return NewError(ErrCodeStorageExecution, "storage", fmt.Sprintf("set operation failed for key '%s': %v", key, err)).
		WithContext("operation", "storage.Set").
		WithContext("key", key).
		WithSeverity("error")
}

// StorageDeleteError creates an error for failed Delete operations
func StorageDeleteError(key string, err error) *Error {
	return NewError(ErrCodeStorageExecution, "storage", fmt.Sprintf("delete operation failed for key '%s': %v", key, err)).
		WithContext("operation", "storage.Delete").
		WithContext("key", key).
		WithSeverity("warning")
}

// StorageListError creates an error for failed List operations
func StorageListError(prefix string, err error) *Error {
	return NewError(ErrCodeStorageExecution, "storage", fmt.Sprintf("list operation failed for prefix '%s': %v", prefix, err)).
		WithContext("operation", "storage.List").
		WithContext("prefix", prefix).
		WithSeverity("warning")
}

// StorageNotFoundError creates an error for key not found scenarios
func StorageNotFoundError(key string) *Error {
	return NewError(ErrCodeStorageNotFound, "storage", fmt.Sprintf("key '%s' not found", key)).
		WithContext("operation", "storage.Get").
		WithContext("key", key).
		WithSeverity("info").
		WithUserMessage(fmt.Sprintf("Key '%s' not found in storage", key))
}

// StorageValidationError creates an error for validation failures
func StorageValidationError(operation, message string) *Error {
	return NewError(ErrCodeStorageValidation, "storage", fmt.Sprintf("%s validation failed: %s", operation, message)).
		WithContext("operation", fmt.Sprintf("storage.%s", operation)).
		WithSeverity("error").
		WithUserMessage("Invalid storage operation parameters")
}

// PluginLoadError creates an error for failed plugin loading
func PluginLoadError(pluginPath string, err error) *Error {
	return NewError(ErrCodePluginError, "storage", fmt.Sprintf("failed to load plugin from '%s': %v", pluginPath, err)).
		WithContext("operation", "storage.LoadPlugin").
		WithContext("plugin_path", pluginPath).
		WithSeverity("critical").
		WithUserMessage("Failed to load storage plugin")
}

// ConfigValidationError creates an error for invalid storage configuration
func ConfigValidationError(provider string, err error) *Error {
	return NewError(ErrCodeStorageValidation, "storage", fmt.Sprintf("configuration validation failed for provider '%s': %v", provider, err)).
		WithContext("operation", "storage.ValidateConfig").
		WithContext("provider", provider).
		WithSeverity("error").
		WithUserMessage("Storage configuration is invalid")
}

// StorageUnavailableError creates an error for unavailable storage backends
func StorageUnavailableError(provider string, err error) *Error {
	return NewError(ErrCodeStorageUnavailable, "storage", fmt.Sprintf("storage provider '%s' is unavailable: %v", provider, err)).
		WithContext("operation", "storage.Connect").
		WithContext("provider", provider).
		WithSeverity("critical").
		WithUserMessage("Storage backend is currently unavailable").
		AsRetryable()
}

// Utility functions for error checking

// IsStorageNotFound checks if an error represents a "key not found" condition
func IsStorageNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our structured error
	if orpheusErr, ok := err.(*Error); ok {
		return orpheusErr.ErrorCode() == ErrCodeStorageNotFound
	}

	// Check if it's a sentinel error
	return errors.Is(err, ErrKeyNotFound)
}

// IsStorageValidationError checks if an error is a storage validation error
func IsStorageValidationError(err error) bool {
	if err == nil {
		return false
	}

	if orpheusErr, ok := err.(*Error); ok {
		return orpheusErr.ErrorCode() == ErrCodeStorageValidation
	}

	return false
}

// IsStorageUnavailable checks if the storage backend is unavailable
func IsStorageUnavailable(err error) bool {
	if err == nil {
		return false
	}

	if orpheusErr, ok := err.(*Error); ok {
		return orpheusErr.ErrorCode() == ErrCodeStorageUnavailable
	}

	// Check sentinel error
	return errors.Is(err, ErrStorageUnavailable)
}
