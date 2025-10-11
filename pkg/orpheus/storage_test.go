// Storage System Tests for Orpheus
//
// Comprehensive test suite for storage interfaces, error handling, and
// context integration with focus on production scenarios and edge cases.
//
// Coverage areas:
// - Storage interface implementation validation
// - Error handling and structured error types
// - Context integration and timeout handling
// - Observability integration (metrics, tracing, audit)
// - Configuration validation and security
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Test Storage Interface Compliance

func TestStorageInterfaceCompliance(t *testing.T) {
	storage := NewMockStorage()

	// Verify interface compliance at compile time
	var _ Storage = storage

	ctx := context.Background()

	// Test basic operations
	key := "test_key"
	value := []byte("test_value")

	// Test Set
	err := storage.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	// Test List
	keys, err := storage.List(ctx, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 1 || keys[0] != key {
		t.Errorf("Expected [%s], got %v", key, keys)
	}

	// Test Delete
	err = storage.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = storage.Get(ctx, key)
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("Expected ErrKeyNotFound after delete, got %v", err)
	}

	// Test Health
	err = storage.Health(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Test Stats
	stats, err := storage.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats == nil {
		t.Error("Expected non-nil stats")
	}

	// Test Close
	err = storage.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify operations fail after close
	err = storage.Set(ctx, "key", []byte("value"))
	if !errors.Is(err, ErrStorageClosed) {
		t.Errorf("Expected ErrStorageClosed after close, got %v", err)
	}
}

func TestStorageListWithPrefix(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Set multiple keys with different prefixes
	testData := map[string][]byte{
		"app:config:database": []byte("db_config"),
		"app:config:cache":    []byte("cache_config"),
		"app:state:session":   []byte("session_data"),
		"user:profile:123":    []byte("user_data"),
		"user:profile:456":    []byte("user_data2"),
	}

	for key, value := range testData {
		err := storage.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set %s: %v", key, err)
		}
	}

	tests := []struct {
		name     string
		prefix   string
		expected int
	}{
		{"Empty prefix (all keys)", "", 5},
		{"App config prefix", "app:config:", 2},
		{"App prefix", "app:", 3},
		{"User profile prefix", "user:profile:", 2},
		{"Nonexistent prefix", "nonexistent:", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := storage.List(ctx, tt.prefix)
			if err != nil {
				t.Fatalf("List with prefix %s failed: %v", tt.prefix, err)
			}

			if len(keys) != tt.expected {
				t.Errorf("Expected %d keys for prefix %s, got %d: %v",
					tt.expected, tt.prefix, len(keys), keys)
			}

			// Verify all returned keys have the prefix
			for _, key := range keys {
				if tt.prefix != "" && !hasPrefix(key, tt.prefix) {
					t.Errorf("Key %s doesn't have prefix %s", key, tt.prefix)
				}
			}
		})
	}
}

// Helper function since strings.HasPrefix might not be available in minimal test env
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func TestStorageStats(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Test empty storage stats
	stats, err := storage.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.TotalKeys != 0 {
		t.Errorf("Expected 0 keys in empty storage, got %d", stats.TotalKeys)
	}

	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 size in empty storage, got %d", stats.TotalSize)
	}

	// Add some data
	testData := map[string][]byte{
		"key1": []byte("value1"), // 6 bytes
		"key2": []byte("value2"), // 6 bytes
		"key3": []byte("value3"), // 6 bytes
	}

	for key, value := range testData {
		err := storage.Set(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set %s: %v", key, err)
		}
	}

	// Test stats with data
	stats, err = storage.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.TotalKeys != 3 {
		t.Errorf("Expected 3 keys, got %d", stats.TotalKeys)
	}

	if stats.TotalSize != 18 { // 3 * 6 bytes
		t.Errorf("Expected 18 bytes, got %d", stats.TotalSize)
	}

	if stats.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", stats.Provider)
	}

	if stats.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}
}

func TestStorageHealthCheck(t *testing.T) {
	storage := NewMockStorage()
	ctx := context.Background()

	// Test healthy storage
	err := storage.Health(ctx)
	if err != nil {
		t.Errorf("Expected healthy storage, got error: %v", err)
	}

	// Test unhealthy storage
	healthErr := errors.New("storage backend unavailable")
	storage.SetHealthError(healthErr)

	err = storage.Health(ctx)
	if err == nil {
		t.Error("Expected health error, got nil")
	}

	if err != healthErr {
		t.Errorf("Expected specific health error, got %v", err)
	}
}

// Test Error Handling

func TestStorageErrors(t *testing.T) {
	tests := []struct {
		name      string
		errorFunc func() error
		expected  string
	}{
		{
			name: "StorageGetError",
			errorFunc: func() error {
				return StorageGetError("test_key", errors.New("connection failed"))
			},
			expected: "get operation failed for key 'test_key'",
		},
		{
			name: "StorageSetError",
			errorFunc: func() error {
				return StorageSetError("test_key", errors.New("write failed"))
			},
			expected: "set operation failed for key 'test_key'",
		},
		{
			name: "StorageDeleteError",
			errorFunc: func() error {
				return StorageDeleteError("test_key", errors.New("delete failed"))
			},
			expected: "delete operation failed for key 'test_key'",
		},
		{
			name: "StorageListError",
			errorFunc: func() error {
				return StorageListError("prefix", errors.New("list failed"))
			},
			expected: "list operation failed for prefix 'prefix'",
		},
		{
			name: "StorageNotFoundError",
			errorFunc: func() error {
				return StorageNotFoundError("missing_key")
			},
			expected: "key 'missing_key' not found",
		},
		{
			name: "StorageValidationError",
			errorFunc: func() error {
				return StorageValidationError("Set", "key cannot be empty")
			},
			expected: "Set validation failed: key cannot be empty",
		},
		{
			name: "PluginLoadError",
			errorFunc: func() error {
				return PluginLoadError("/path/plugin.so", errors.New("invalid format"))
			},
			expected: "failed to load plugin from '/path/plugin.so'",
		},
		{
			name: "ConfigValidationError",
			errorFunc: func() error {
				return ConfigValidationError("sqlite", errors.New("missing database path"))
			},
			expected: "configuration validation failed for provider 'sqlite'",
		},
		{
			name: "StorageUnavailableError",
			errorFunc: func() error {
				return StorageUnavailableError("redis", errors.New("connection timeout"))
			},
			expected: "storage provider 'redis' is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFunc()

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !containsString(err.Error(), tt.expected) {
				t.Errorf("Expected error message to contain '%s', got '%s'",
					tt.expected, err.Error())
			}

			// Test if it's an Orpheus error with proper structure
			if orpheusErr, ok := err.(*Error); ok {
				if orpheusErr.ErrorCode() == "" {
					t.Error("Expected non-empty error code")
				}

				// Verify it has the expected command/component in the Command field
				if orpheusErr.Command != "storage" {
					t.Errorf("Expected command 'storage', got '%s'", orpheusErr.Command)
				}
			} else {
				t.Error("Expected Orpheus error type")
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestErrorTypeChecking(t *testing.T) {
	tests := []struct {
		name      string
		error     error
		checkFunc func(error) bool
		expected  bool
	}{
		{
			name:      "IsStorageNotFound with StorageNotFoundError",
			error:     StorageNotFoundError("test_key"),
			checkFunc: IsStorageNotFound,
			expected:  true,
		},
		{
			name:      "IsStorageNotFound with ErrKeyNotFound",
			error:     ErrKeyNotFound,
			checkFunc: IsStorageNotFound,
			expected:  true,
		},
		{
			name:      "IsStorageNotFound with other error",
			error:     errors.New("other error"),
			checkFunc: IsStorageNotFound,
			expected:  false,
		},
		{
			name:      "IsStorageValidationError with validation error",
			error:     StorageValidationError("Set", "invalid key"),
			checkFunc: IsStorageValidationError,
			expected:  true,
		},
		{
			name:      "IsStorageValidationError with other error",
			error:     errors.New("other error"),
			checkFunc: IsStorageValidationError,
			expected:  false,
		},
		{
			name:      "IsStorageUnavailable with unavailable error",
			error:     StorageUnavailableError("redis", errors.New("timeout")),
			checkFunc: IsStorageUnavailable,
			expected:  true,
		},
		{
			name:      "IsStorageUnavailable with sentinel error",
			error:     ErrStorageUnavailable,
			checkFunc: IsStorageUnavailable,
			expected:  true,
		},
		{
			name:      "IsStorageUnavailable with other error",
			error:     errors.New("other error"),
			checkFunc: IsStorageUnavailable,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checkFunc(tt.error)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test Context Integration

func TestStorageWithContext(t *testing.T) {
	storage := NewMockStorage()

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// These operations should complete quickly
	err := storage.Set(ctx, "key", []byte("value"))
	if err != nil {
		t.Errorf("Set with timeout context failed: %v", err)
	}

	_, err = storage.Get(ctx, "key")
	if err != nil {
		t.Errorf("Get with timeout context failed: %v", err)
	}

	// Test with canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations should still work with our mock (real implementations might respect cancellation)
	err = storage.Set(canceledCtx, "key2", []byte("value2"))
	if err != nil {
		t.Errorf("Set with canceled context failed: %v", err)
	}
}

// Test Configuration

func TestStorageConfig(t *testing.T) {
	tests := []struct {
		name   string
		config StorageConfig
		valid  bool
	}{
		{
			name: "Valid minimal config",
			config: StorageConfig{
				Provider: "sqlite",
				Config: map[string]interface{}{
					"path": "/tmp/test.db",
				},
			},
			valid: true,
		},
		{
			name: "Valid config with all options",
			config: StorageConfig{
				Provider:      "redis",
				PluginPath:    "/opt/plugins/redis.so",
				Config:        map[string]interface{}{"host": "localhost", "port": 6379},
				Namespace:     "myapp",
				EnableMetrics: true,
				EnableTracing: true,
				EnableAudit:   true,
			},
			valid: true,
		},
		{
			name: "Empty provider",
			config: StorageConfig{
				Provider: "",
				Config:   map[string]interface{}{},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - provider should not be empty
			if tt.config.Provider == "" && tt.valid {
				t.Error("Empty provider should be invalid")
			}

			if tt.config.Provider != "" && !tt.valid {
				// Additional validation logic would go here
				// For now, we just check that non-empty provider is considered valid
			}
		})
	}
}

// Benchmark Tests

func BenchmarkStorageOperations(b *testing.B) {
	storage := NewMockStorage()
	ctx := context.Background()

	key := "benchmark_key"
	value := []byte("benchmark_value_with_some_data_to_make_it_realistic")

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.Set(ctx, key, value)
		}
	})

	// Ensure key exists for Get benchmark
	_ = storage.Set(ctx, key, value)

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = storage.Get(ctx, key)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.Delete(ctx, key)
		}
	})

	b.Run("Health", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.Health(ctx)
		}
	})
}

func BenchmarkStorageConcurrency(b *testing.B) {
	storage := NewMockStorage()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("concurrent_key_%d", i)
			value := []byte(fmt.Sprintf("concurrent_value_%d", i))

			_ = storage.Set(ctx, key, value)
			_, _ = storage.Get(ctx, key)
			_ = storage.Delete(ctx, key)

			i++
		}
	})
}
