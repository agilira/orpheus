// Storage System for Orpheus CLI Framework
//
// This package provides pluggable storage capabilities for CLI applications,
// enabling persistent state management with zero-overhead when unused.
//
// Features:
// - Interface-based architecture for maximum flexibility
// - Plugin system with .so dynamic loading
// - Full observability integration (metrics, tracing, audit)
// - Structured error handling with context
// - Security-hardened key validation
// - Zero performance overhead when not configured
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"time"
)

// Storage provides a pluggable key-value storage interface for CLI applications.
// Implementations must be thread-safe and support context-based cancellation.
// All operations should integrate with observability systems when available.
type Storage interface {
	// Get retrieves a value by key. Returns ErrKeyNotFound if key doesn't exist.
	// The context should be used for tracing, timeout, and cancellation.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value with the given key. Overwrites existing values.
	// The context should be used for tracing, timeout, and cancellation.
	Set(ctx context.Context, key string, value []byte) error

	// Delete removes a key and its value. No error if key doesn't exist.
	// The context should be used for tracing, timeout, and cancellation.
	Delete(ctx context.Context, key string) error

	// List returns all keys matching the given prefix, sorted lexicographically.
	// Empty prefix returns all keys. Returns empty slice if no matches found.
	List(ctx context.Context, prefix string) ([]string, error)

	// Health performs a health check on the storage backend.
	// Should return quickly and indicate if the storage is operational.
	Health(ctx context.Context) error

	// Stats returns operational statistics for monitoring and debugging.
	// Returns nil if statistics are not available or supported.
	Stats(ctx context.Context) (*StorageStats, error)

	// Close releases any resources held by the storage implementation.
	// After calling Close, the Storage instance should not be used.
	Close() error
}

// StorageStats provides operational metrics for storage backends.
// All fields are optional and may be zero if not supported.
type StorageStats struct {
	// TotalKeys is the number of keys currently stored
	TotalKeys int64 `json:"total_keys"`

	// TotalSize is the approximate total size in bytes
	TotalSize int64 `json:"total_size_bytes"`

	// Operations counters since startup or reset
	GetOperations    int64 `json:"get_operations"`
	SetOperations    int64 `json:"set_operations"`
	DeleteOperations int64 `json:"delete_operations"`
	ListOperations   int64 `json:"list_operations"`

	// Error counters
	GetErrors    int64 `json:"get_errors"`
	SetErrors    int64 `json:"set_errors"`
	DeleteErrors int64 `json:"delete_errors"`
	ListErrors   int64 `json:"list_errors"`

	// Performance metrics (averages)
	AvgGetLatency    time.Duration `json:"avg_get_latency_ns"`
	AvgSetLatency    time.Duration `json:"avg_set_latency_ns"`
	AvgDeleteLatency time.Duration `json:"avg_delete_latency_ns"`
	AvgListLatency   time.Duration `json:"avg_list_latency_ns"`

	// Provider-specific information
	Provider string                 `json:"provider"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Uptime   time.Duration          `json:"uptime_ns"`
}

// StoragePlugin defines the interface that storage providers must implement
// to be loaded as plugins. This enables dynamic loading of storage backends
// without compile-time dependencies.
type StoragePlugin interface {
	// Name returns the unique identifier for this storage provider
	Name() string

	// Version returns the version string for this provider
	Version() string

	// Description returns a human-readable description of the provider
	Description() string

	// New creates a new Storage instance with the given configuration.
	// The config map contains provider-specific configuration options.
	New(config map[string]interface{}) (Storage, error)

	// Validate checks if the provided configuration is valid for this provider.
	// Should return a descriptive error if the configuration is invalid.
	Validate(config map[string]interface{}) error

	// DefaultConfig returns the default configuration for this provider.
	// Used for documentation and initialization.
	DefaultConfig() map[string]interface{}
}

// StorageConfig holds the configuration for storage initialization
type StorageConfig struct {
	// Provider specifies which storage provider to use
	Provider string `json:"provider" yaml:"provider"`

	// PluginPath is the filesystem path to the provider plugin (.so file)
	// If empty, looks for built-in providers first, then standard plugin locations
	PluginPath string `json:"plugin_path,omitempty" yaml:"plugin_path,omitempty"`

	// Config contains provider-specific configuration options
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	// Namespace provides key isolation when multiple apps share the same storage
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// EnableMetrics controls whether to collect and expose storage metrics
	EnableMetrics bool `json:"enable_metrics,omitempty" yaml:"enable_metrics,omitempty"`

	// EnableTracing controls whether to create tracing spans for storage operations
	EnableTracing bool `json:"enable_tracing,omitempty" yaml:"enable_tracing,omitempty"`

	// EnableAudit controls whether to log storage operations for security auditing
	EnableAudit bool `json:"enable_audit,omitempty" yaml:"enable_audit,omitempty"`
}
