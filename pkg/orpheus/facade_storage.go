// facade_storage.go: the storage API under its v1.11.0 names
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"github.com/agilira/orpheus/observe"
	"github.com/agilira/orpheus/storage"
)

// Storage types, see package storage.
type (
	Storage              = storage.Storage
	StorageStats         = storage.Stats
	StoragePlugin        = storage.Plugin
	StorageConfig        = storage.Config
	PluginManager        = storage.PluginManager
	LoadedPlugin         = storage.LoadedPlugin
	PluginSecurityConfig = storage.PluginSecurityConfig
)

// Storage sentinel errors: the very values of package storage, so that
// errors.Is matches either name.
var (
	ErrKeyNotFound          = storage.ErrKeyNotFound
	ErrKeyEmpty             = storage.ErrKeyEmpty
	ErrKeyTooLong           = storage.ErrKeyTooLong
	ErrKeyInvalid           = storage.ErrKeyInvalid
	ErrValueTooLarge        = storage.ErrValueTooLarge
	ErrStorageNotConfigured = storage.ErrNotConfigured
	ErrStorageUnavailable   = storage.ErrUnavailable
	ErrStorageClosed        = storage.ErrClosed
	ErrPluginNotFound       = storage.ErrPluginNotFound
	ErrPluginInvalid        = storage.ErrPluginInvalid
	ErrConfigInvalid        = storage.ErrConfigInvalid
)

// DefaultPluginSecurityConfig calls storage.DefaultPluginSecurityConfig.
func DefaultPluginSecurityConfig() *PluginSecurityConfig {
	return storage.DefaultPluginSecurityConfig()
}

// NewPluginManager calls storage.NewPluginManager.
func NewPluginManager(logger observe.Logger, securityConfig *PluginSecurityConfig) *PluginManager {
	return storage.NewPluginManager(logger, securityConfig)
}

// StorageGetError calls storage.GetError.
func StorageGetError(key string, err error) *Error { return storage.GetError(key, err) }

// StorageSetError calls storage.SetError.
func StorageSetError(key string, err error) *Error { return storage.SetError(key, err) }

// StorageDeleteError calls storage.DeleteError.
func StorageDeleteError(key string, err error) *Error { return storage.DeleteError(key, err) }

// StorageListError calls storage.ListError.
func StorageListError(prefix string, err error) *Error { return storage.ListError(prefix, err) }

// StorageNotFoundError calls storage.NotFoundError.
func StorageNotFoundError(key string) *Error { return storage.NotFoundError(key) }

// StorageValidationError calls storage.ValidationError.
func StorageValidationError(operation, message string) *Error {
	return storage.ValidationError(operation, message)
}

// PluginLoadError calls storage.PluginLoadError.
func PluginLoadError(pluginPath string, err error) *Error {
	return storage.PluginLoadError(pluginPath, err)
}

// ConfigValidationError calls storage.ConfigValidationError.
func ConfigValidationError(provider string, err error) *Error {
	return storage.ConfigValidationError(provider, err)
}

// StorageUnavailableError calls storage.UnavailableError.
func StorageUnavailableError(provider string, err error) *Error {
	return storage.UnavailableError(provider, err)
}

// IsStorageNotFound calls storage.IsNotFound.
func IsStorageNotFound(err error) bool { return storage.IsNotFound(err) }

// IsStorageValidationError calls storage.IsValidationError.
func IsStorageValidationError(err error) bool { return storage.IsValidationError(err) }

// IsStorageUnavailable calls storage.IsUnavailable.
func IsStorageUnavailable(err error) bool { return storage.IsUnavailable(err) }
