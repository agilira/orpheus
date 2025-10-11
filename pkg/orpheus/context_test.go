// context_test.go: Orpheus application framework context tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestContextArguments(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{"arg1", "arg2", "arg3"},
	}

	if ctx.ArgCount() != 3 {
		t.Errorf("expected 3 args, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "arg1" {
		t.Errorf("expected arg[0] = 'arg1', got '%s'", ctx.GetArg(0))
	}

	if ctx.GetArg(1) != "arg2" {
		t.Errorf("expected arg[1] = 'arg2', got '%s'", ctx.GetArg(1))
	}

	if ctx.GetArg(2) != "arg3" {
		t.Errorf("expected arg[2] = 'arg3', got '%s'", ctx.GetArg(2))
	}
}

func TestContextOutOfBoundsArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{"arg1"},
	}

	// Test negative index
	if ctx.GetArg(-1) != "" {
		t.Errorf("expected empty string for negative index, got '%s'", ctx.GetArg(-1))
	}

	// Test index beyond bounds
	if ctx.GetArg(5) != "" {
		t.Errorf("expected empty string for out of bounds index, got '%s'", ctx.GetArg(5))
	}

	// Test exactly at bounds
	if ctx.GetArg(1) != "" {
		t.Errorf("expected empty string for index at bounds, got '%s'", ctx.GetArg(1))
	}
}

func TestContextEmptyArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: []string{},
	}

	if ctx.ArgCount() != 0 {
		t.Errorf("expected 0 args, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "" {
		t.Errorf("expected empty string for empty args, got '%s'", ctx.GetArg(0))
	}
}

func TestContextNilArgs(t *testing.T) {
	ctx := &orpheus.Context{
		Args: nil,
	}

	if ctx.ArgCount() != 0 {
		t.Errorf("expected 0 args for nil slice, got %d", ctx.ArgCount())
	}

	if ctx.GetArg(0) != "" {
		t.Errorf("expected empty string for nil args, got '%s'", ctx.GetArg(0))
	}
}

func TestContextFlags(t *testing.T) {
	ctx := &orpheus.Context{}

	// Test flag methods return default values when no flags are set
	if ctx.GetFlag("test") != nil {
		t.Errorf("expected nil for non-existent flag, got %v", ctx.GetFlag("test"))
	}

	if ctx.GetFlagString("test") != "" {
		t.Errorf("expected empty string for non-existent string flag, got '%s'", ctx.GetFlagString("test"))
	}

	if ctx.GetFlagBool("test") != false {
		t.Errorf("expected false for non-existent bool flag, got %v", ctx.GetFlagBool("test"))
	}

	if ctx.GetFlagInt("test") != 0 {
		t.Errorf("expected 0 for non-existent int flag, got %d", ctx.GetFlagInt("test"))
	}
}

func TestContextGlobalFlags(t *testing.T) {
	ctx := &orpheus.Context{}

	// Test global flag methods return default values when no flags are set
	if ctx.GetGlobalFlag("test") != nil {
		t.Errorf("expected nil for non-existent global flag, got %v", ctx.GetGlobalFlag("test"))
	}

	if ctx.GetGlobalFlagBool("test") != false {
		t.Errorf("expected false for non-existent global bool flag, got %v", ctx.GetGlobalFlagBool("test"))
	}

	// Test additional global flag methods for non-existent flags
	if ctx.GetGlobalFlagString("test") != "" {
		t.Errorf("expected empty string for non-existent global string flag, got %v", ctx.GetGlobalFlagString("test"))
	}

	if ctx.GetGlobalFlagInt("test") != 0 {
		t.Errorf("expected 0 for non-existent global int flag, got %v", ctx.GetGlobalFlagInt("test"))
	}

	if ctx.GlobalFlagChanged("test") != false {
		t.Errorf("expected false for non-existent global flag changed, got %v", ctx.GlobalFlagChanged("test"))
	}
}

// TestContextObservabilityGetters tests context observability method getters
func TestContextObservabilityGetters(t *testing.T) {
	// Test with nil app
	ctx := &orpheus.Context{App: nil}

	if ctx.Logger() != nil {
		t.Error("expected nil logger when app is nil")
	}

	if ctx.AuditLogger() != nil {
		t.Error("expected nil audit logger when app is nil")
	}

	if ctx.Tracer() != nil {
		t.Error("expected nil tracer when app is nil")
	}

	if ctx.MetricsCollector() != nil {
		t.Error("expected nil metrics collector when app is nil")
	}

	// Test with app but no observability components set
	app := orpheus.New("testapp")
	ctx = &orpheus.Context{App: app}

	if ctx.Logger() != nil {
		t.Error("expected nil logger when not configured in app")
	}

	if ctx.AuditLogger() != nil {
		t.Error("expected nil audit logger when not configured in app")
	}

	if ctx.Tracer() != nil {
		t.Error("expected nil tracer when not configured in app")
	}

	if ctx.MetricsCollector() != nil {
		t.Error("expected nil metrics collector when not configured in app")
	}
}

// TestContextFlagChangedEdgeCases tests edge cases for FlagChanged method
func TestContextFlagChangedEdgeCases(t *testing.T) {
	// Test with nil flags
	ctx := &orpheus.Context{
		Args:  []string{"test"},
		Flags: nil,
	}

	// Test FlagChanged with nil flags - should not panic and return false
	if ctx.FlagChanged("debug") {
		t.Error("FlagChanged should return false with nil flags")
	}

	if ctx.FlagChanged("nonexistent") {
		t.Error("FlagChanged should return false for nonexistent flag with nil flags")
	}

	// Test edge case: empty flag name
	if ctx.FlagChanged("") {
		t.Error("FlagChanged should return false for empty flag name")
	}
}

func TestContextGetFlagFloat64AndStringSlice(t *testing.T) {
	// Test with nil flags (edge case coverage)
	ctx := &orpheus.Context{
		Args:  []string{"test"},
		Flags: nil,
	}

	// Test GetFlagFloat64 with nil flags
	defaultRate := ctx.GetFlagFloat64("rate")
	if defaultRate != 0.0 {
		t.Errorf("Expected default rate 0.0 with nil flags, got %f", defaultRate)
	}

	// Test GetFlagStringSlice with nil flags
	defaultTags := ctx.GetFlagStringSlice("tags")
	if len(defaultTags) != 0 {
		t.Errorf("Expected empty slice with nil flags, got %v", defaultTags)
	}

	// Verify the methods don't panic with nil flags
	intVal := ctx.GetFlagInt("count")
	if intVal != 0 {
		t.Errorf("Expected 0 for int flag with nil flags, got %d", intVal)
	}

	flagVal := ctx.GetFlag("name")
	if flagVal != nil {
		t.Errorf("Expected nil for flag with nil flags, got %v", flagVal)
	}
}

// TestContextStorageOperations tests storage-related context methods for security and safety
func TestContextStorageOperations(t *testing.T) {
	t.Run("SetStorage_WithValidStorage", func(t *testing.T) {
		// Test setting a valid storage implementation
		ctx := &orpheus.Context{}
		mockStorage := &mockStorage{data: make(map[string][]byte)}

		// This should not panic and should properly store the reference
		ctx.SetStorage(mockStorage)

		// Verify the storage was set (check via interface equality)
		if ctx.Storage() == nil {
			t.Error("Expected storage to be set correctly")
		}
	})

	t.Run("SetStorage_WithNilStorage", func(t *testing.T) {
		// Test setting nil storage (should be allowed for cleanup)
		ctx := &orpheus.Context{}

		// Set a storage first
		mockStorage := &mockStorage{data: make(map[string][]byte)}
		ctx.SetStorage(mockStorage)

		// Now set to nil (cleanup scenario)
		ctx.SetStorage(nil)

		// Verify the storage was cleared
		if ctx.Storage() != nil {
			t.Error("Expected storage to be nil after setting to nil")
		}
	})

	t.Run("SetStorage_OverwriteExisting", func(t *testing.T) {
		// Test overwriting existing storage (security concern)
		ctx := &orpheus.Context{}

		// Set initial storage
		storage1 := &mockStorage{data: make(map[string][]byte)}
		storage1.data["key1"] = []byte("value1")
		ctx.SetStorage(storage1)

		// Overwrite with new storage
		storage2 := &mockStorage{data: make(map[string][]byte)}
		storage2.data["key2"] = []byte("value2")
		ctx.SetStorage(storage2)

		// Verify new storage replaced old one
		if ctx.Storage() == nil {
			t.Error("Expected storage to be set after replacement")
		}

		// Verify old storage data is no longer accessible
		result, err := ctx.Storage().Get(context.Background(), "key1")
		if err == nil || len(result) > 0 {
			t.Error("Expected old storage data to be inaccessible after replacement")
		}
	})

	t.Run("RequireStorage_WithStoragePresent", func(t *testing.T) {
		// Test RequireStorage when storage is available
		ctx := &orpheus.Context{}
		mockStorage := &mockStorage{data: make(map[string][]byte)}
		ctx.SetStorage(mockStorage)

		// This should return the storage without error
		storage, err := ctx.RequireStorage()
		if err != nil {
			t.Errorf("Expected no error when storage is present, got: %v", err)
		}

		if storage == nil {
			t.Error("Expected RequireStorage to return the storage instance")
		}
	})

	t.Run("RequireStorage_WithoutStorage", func(t *testing.T) {
		// Test RequireStorage when no storage is configured (security check)
		ctx := &orpheus.Context{}

		// This should return an error
		storage, err := ctx.RequireStorage()
		if err == nil {
			t.Error("Expected error when no storage is configured")
		}

		if storage != nil {
			t.Error("Expected nil storage when error is returned")
		}

		// Verify error contains expected message (storage unavailable scenario)
		if err.Error() == "" {
			t.Error("Expected non-empty error message when storage is not configured")
		}
	})
}

// mockStorage implements orpheus.Storage interface for testing
type mockStorage struct {
	data   map[string][]byte
	closed bool
}

func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	if m.closed {
		return nil, fmt.Errorf("storage is closed")
	}
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockStorage) Set(ctx context.Context, key string, value []byte) error {
	if m.closed {
		return fmt.Errorf("storage is closed")
	}
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	m.data[key] = value
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.closed {
		return fmt.Errorf("storage is closed")
	}
	if _, exists := m.data[key]; !exists {
		return fmt.Errorf("key not found")
	}
	delete(m.data, key)
	return nil
}

func (m *mockStorage) List(ctx context.Context, prefix string) ([]string, error) {
	if m.closed {
		return nil, fmt.Errorf("storage is closed")
	}
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockStorage) Health(ctx context.Context) error {
	if m.closed {
		return fmt.Errorf("storage is closed")
	}
	return nil
}

func (m *mockStorage) Stats(ctx context.Context) (*orpheus.StorageStats, error) {
	if m.closed {
		return nil, fmt.Errorf("storage is closed")
	}
	return &orpheus.StorageStats{
		TotalKeys: int64(len(m.data)),
		Provider:  "mock",
	}, nil
}

func (m *mockStorage) Close() error {
	m.closed = true
	return nil
}
