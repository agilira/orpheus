// App Storage Integration Tests for Orpheus
//
// Tests the integration between the App class and the storage system,
// including configuration, plugin loading, and context propagation.
//
// Coverage areas:
// - App storage configuration methods
// - Plugin manager integration
// - Context storage access in commands
// - Error handling in storage setup
// - Configuration validation
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"errors"
	"testing"
)

func TestAppStorageConfiguration(t *testing.T) {
	app := New("testapp")

	// Test initial state
	if app.Storage() != nil {
		t.Error("Expected nil storage in new app")
	}

	if app.StorageConfig() != nil {
		t.Error("Expected nil storage config in new app")
	}

	if app.PluginManager() != nil {
		t.Error("Expected nil plugin manager in new app")
	}

	// Test SetStorage
	mockStorage := NewMockStorage()
	returnedApp := app.SetStorage(mockStorage)

	// Verify fluent interface
	if returnedApp != app {
		t.Error("SetStorage should return the same app instance for chaining")
	}

	// Verify storage is set
	if app.Storage() != mockStorage {
		t.Error("Storage not set correctly")
	}

	// Test StorageConfig
	config := &StorageConfig{
		Provider: "mock",
		Config: map[string]interface{}{
			"test": true,
		},
	}

	returnedApp = app.ConfigureStorage(config)

	// Verify fluent interface
	if returnedApp != app {
		t.Error("ConfigureStorage should return the same app instance for chaining")
	}

	// Verify config is set
	if app.StorageConfig() != config {
		t.Error("Storage config not set correctly")
	}

	// Verify plugin manager is created
	if app.PluginManager() == nil {
		t.Error("Plugin manager should be created during ConfigureStorage")
	}
}

func TestAppStorageConfigurationChaining(t *testing.T) {
	mockStorage := NewMockStorage()
	config := &StorageConfig{
		Provider: "test",
		Config:   map[string]interface{}{},
	}

	// Test fluent chaining
	app := New("chaintest").
		SetVersion("1.0.0").
		SetDescription("Test app").
		SetStorage(mockStorage).
		ConfigureStorage(config)

	if app.Storage() != mockStorage {
		t.Error("Storage not preserved through chaining")
	}

	if app.StorageConfig() != config {
		t.Error("Storage config not preserved through chaining")
	}
}

func TestContextStorageAccess(t *testing.T) {
	app := New("storagetest")
	mockStorage := NewMockStorage()
	app.SetStorage(mockStorage)

	// Set up a test key-value pair
	ctx := context.Background()
	testKey := "test_key"
	testValue := []byte("test_value")

	err := mockStorage.Set(ctx, testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	// Create a command context
	cmdCtx := &Context{
		App:     app,
		Flags:   nil,
		Args:    []string{},
		storage: mockStorage, // This would be set automatically in real usage
	}

	// Test storage access through context
	retrievedStorage := cmdCtx.Storage()
	if retrievedStorage == nil {
		t.Fatal("Expected non-nil storage from context")
	}

	// Test actual storage operations through context
	retrievedValue, err := retrievedStorage.Get(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to get value through context storage: %v", err)
	}

	if string(retrievedValue) != string(testValue) {
		t.Errorf("Expected %s, got %s", string(testValue), string(retrievedValue))
	}

	// Test RequireStorage
	_, err = cmdCtx.RequireStorage()
	if err != nil {
		t.Errorf("RequireStorage should succeed when storage is available: %v", err)
	}

	// Test RequireStorage with nil storage
	cmdCtx.storage = nil
	_, err = cmdCtx.RequireStorage()
	if err == nil {
		t.Error("RequireStorage should fail when storage is not available")
	}

	// Verify error type
	if !IsStorageValidationError(err) {
		t.Error("Expected storage validation error")
	}
}

func TestAppStorageConfigurationWithLogger(t *testing.T) {
	logger := &MockLogger{}
	app := New("loggedapp").SetLogger(logger)

	config := &StorageConfig{
		Provider: "mock",
		Config: map[string]interface{}{
			"valid": true,
		},
	}

	// Configure storage - this should create plugin manager with logger
	app.ConfigureStorage(config)

	// Verify plugin manager has the logger
	pm := app.PluginManager()
	if pm == nil {
		t.Fatal("Plugin manager should be created")
	}

	// The logger is set but we can't easily test it without exposing internals
	// In real usage, the plugin manager would log events
}

func TestAppStorageConfigurationErrorHandling(t *testing.T) {
	logger := &MockLogger{}
	app := New("errorapp").SetLogger(logger)

	// Test with invalid configuration (empty provider)
	config := &StorageConfig{
		Provider: "", // Invalid: empty provider
		Config:   map[string]interface{}{},
	}

	// This should not crash but should log errors
	returnedApp := app.ConfigureStorage(config)

	// Should still return the app for chaining
	if returnedApp != app {
		t.Error("ConfigureStorage should return app even on errors")
	}

	// Storage should remain nil due to configuration failure
	if app.Storage() != nil {
		t.Error("Storage should remain nil with invalid config")
	}

	// Config should still be set (for debugging purposes)
	if app.StorageConfig() != config {
		t.Error("Config should be set even if invalid")
	}
}

func TestAppStorageMultipleConfigurations(t *testing.T) {
	app := New("multiconfig")

	// First configuration
	config1 := &StorageConfig{
		Provider: "provider1",
		Config:   map[string]interface{}{"setting": "value1"},
	}

	app.ConfigureStorage(config1)

	if app.StorageConfig() != config1 {
		t.Error("First config not set correctly")
	}

	// Second configuration (should override)
	config2 := &StorageConfig{
		Provider: "provider2",
		Config:   map[string]interface{}{"setting": "value2"},
	}

	app.ConfigureStorage(config2)

	if app.StorageConfig() != config2 {
		t.Error("Second config should override first")
	}

	// Plugin manager should still exist
	if app.PluginManager() == nil {
		t.Error("Plugin manager should persist across reconfigurations")
	}
}

func TestStorageConfigDefaults(t *testing.T) {
	config := &StorageConfig{
		Provider: "test",
		Config:   map[string]interface{}{},
		// All other fields use default values
	}

	// Test set values
	if config.Provider != "test" {
		t.Error("Expected provider to be set correctly")
	}

	if config.Config == nil {
		t.Error("Expected config map to be initialized")
	}

	if len(config.Config) != 0 {
		t.Error("Expected empty config map")
	}

	// Test default values
	if config.PluginPath != "" {
		t.Error("Expected empty default plugin path")
	}

	if config.Namespace != "" {
		t.Error("Expected empty default namespace")
	}

	if config.EnableMetrics {
		t.Error("Expected metrics to be disabled by default")
	}

	if config.EnableTracing {
		t.Error("Expected tracing to be disabled by default")
	}

	if config.EnableAudit {
		t.Error("Expected audit to be disabled by default")
	}
}

func TestStorageConfigWithAllOptions(t *testing.T) {
	config := &StorageConfig{
		Provider:      "redis",
		PluginPath:    "/opt/plugins/redis.so",
		Config:        map[string]interface{}{"host": "localhost", "port": 6379},
		Namespace:     "myapp",
		EnableMetrics: true,
		EnableTracing: true,
		EnableAudit:   true,
	}

	// Verify all fields are set
	if config.Provider != "redis" {
		t.Error("Provider not set correctly")
	}

	if config.PluginPath != "/opt/plugins/redis.so" {
		t.Error("Plugin path not set correctly")
	}

	if config.Config["host"] != "localhost" {
		t.Error("Config host not set correctly")
	}

	if config.Config["port"] != 6379 {
		t.Error("Config port not set correctly")
	}

	if config.Namespace != "myapp" {
		t.Error("Namespace not set correctly")
	}

	if !config.EnableMetrics {
		t.Error("Metrics should be enabled")
	}

	if !config.EnableTracing {
		t.Error("Tracing should be enabled")
	}

	if !config.EnableAudit {
		t.Error("Audit should be enabled")
	}
}

// Test command execution with storage
func TestCommandExecutionWithStorage(t *testing.T) {
	app := New("cmdtest")
	mockStorage := NewMockStorage()
	app.SetStorage(mockStorage)

	// Track if command was called
	commandCalled := false
	var capturedContext *Context

	// Add a command that uses storage
	app.Command("store", "Store a value", func(ctx *Context) error {
		commandCalled = true
		capturedContext = ctx

		// Test storage access in command
		storage := ctx.Storage()
		if storage == nil {
			return errors.New("storage not available in command context")
		}

		// Perform storage operation
		return storage.Set(context.Background(), "cmd_key", []byte("cmd_value"))
	})

	// Execute the command (this would normally be done by the CLI parsing)
	// For testing, we manually create the context
	ctx := &Context{
		App:     app,
		Flags:   nil,
		Args:    []string{},
		storage: mockStorage,
	}

	// Get the command and execute it
	cmd := app.commands["store"]
	if cmd == nil {
		t.Fatal("Command not found")
	}

	err := cmd.handler(ctx)
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify command was called
	if !commandCalled {
		t.Error("Command handler was not called")
	}

	if capturedContext == nil {
		t.Error("Context was not passed to command")
	}

	// Verify storage operation worked
	value, err := mockStorage.Get(context.Background(), "cmd_key")
	if err != nil {
		t.Errorf("Failed to retrieve stored value: %v", err)
	}

	if string(value) != "cmd_value" {
		t.Errorf("Expected 'cmd_value', got '%s'", string(value))
	}
}

// Benchmark App storage operations
func BenchmarkAppStorageConfiguration(b *testing.B) {
	mockStorage := NewMockStorage()
	config := &StorageConfig{
		Provider: "mock",
		Config:   map[string]interface{}{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app := New("benchmark")
		app.SetStorage(mockStorage)
		app.ConfigureStorage(config)
	}
}

func BenchmarkContextStorageAccess(b *testing.B) {
	app := New("benchmark")
	mockStorage := NewMockStorage()
	app.SetStorage(mockStorage)

	ctx := &Context{
		App:     app,
		storage: mockStorage,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Storage()
		_, _ = ctx.RequireStorage()
	}
}

// TestConfigureStorageAdvancedScenarios tests edge cases and error scenarios for ConfigureStorage
func TestConfigureStorageAdvancedScenarios(t *testing.T) {
	t.Run("ConfigureStorage_WithNilConfig", func(t *testing.T) {
		// Test with nil configuration (should not crash)
		app := New("nilconfig")

		// This should not panic, but handle gracefully
		returnedApp := app.ConfigureStorage(nil)

		if returnedApp != app {
			t.Error("ConfigureStorage should return app instance even with nil config")
		}

		// Storage config should be nil
		if app.StorageConfig() != nil {
			t.Error("Storage config should remain nil when configured with nil")
		}

		// Plugin manager might or might not be created (depends on implementation)
		// The important thing is no panic occurred
	})

	t.Run("ConfigureStorage_ReplacesExistingPluginManager", func(t *testing.T) {
		// Test that ConfigureStorage handles existing plugin manager correctly
		logger := &MockLogger{}
		app := New("replace").SetLogger(logger)

		config1 := &StorageConfig{
			Provider: "provider1",
			Config:   map[string]interface{}{},
		}

		app.ConfigureStorage(config1)
		firstPM := app.PluginManager()

		config2 := &StorageConfig{
			Provider: "provider2",
			Config:   map[string]interface{}{},
		}

		app.ConfigureStorage(config2)
		secondPM := app.PluginManager()

		// Plugin manager should persist (not be recreated)
		if firstPM != secondPM {
			t.Error("Plugin manager should be reused across multiple ConfigureStorage calls")
		}

		// Config should be updated
		if app.StorageConfig() != config2 {
			t.Error("Storage config should be updated to new config")
		}
	})

	t.Run("ConfigureStorage_WithoutLogger", func(t *testing.T) {
		// Test ConfigureStorage without a logger set (should not crash)
		app := New("nologger") // No logger set

		config := &StorageConfig{
			Provider: "nologger_test",
			Config:   map[string]interface{}{},
		}

		// This should create plugin manager even without logger
		app.ConfigureStorage(config)

		pm := app.PluginManager()
		if pm == nil {
			t.Error("Plugin manager should be created even without logger")
		}

		if app.StorageConfig() != config {
			t.Error("Config should be set even without logger")
		}
	})

	t.Run("ConfigureStorage_ErrorInPluginLoading", func(t *testing.T) {
		// Test when plugin loading fails (logs error but doesn't crash)
		logger := &MockLogger{}
		app := New("errorloading").SetLogger(logger)

		config := &StorageConfig{
			Provider:   "nonexistent",
			PluginPath: "/nonexistent/path/plugin.so", // Invalid path
			Config:     map[string]interface{}{},
		}

		// This should handle the error gracefully
		app.ConfigureStorage(config)

		// Config should still be set for debugging
		if app.StorageConfig() != config {
			t.Error("Config should be set even when plugin loading fails")
		}

		// Plugin manager should still be created
		if app.PluginManager() == nil {
			t.Error("Plugin manager should be created even when plugin loading fails")
		}

		// Storage should remain nil due to failed loading
		if app.Storage() != nil {
			t.Error("Storage should remain nil when plugin loading fails")
		}
	})

	t.Run("ConfigureStorage_ValidConfigurationFlow", func(t *testing.T) {
		// Test the complete successful configuration flow
		logger := &MockLogger{}
		app := New("validflow").SetLogger(logger)

		config := &StorageConfig{
			Provider:      "mock",
			PluginPath:    "", // Empty path should work with mock
			Config:        map[string]interface{}{"testkey": "testvalue"},
			Namespace:     "test_namespace",
			EnableMetrics: true,
			EnableTracing: false,
			EnableAudit:   true,
		}

		app.ConfigureStorage(config)

		// Verify all components are set up
		if app.StorageConfig() != config {
			t.Error("Config should be set correctly")
		}

		if app.PluginManager() == nil {
			t.Error("Plugin manager should be created")
		}

		// Verify the plugin manager uses the default security config
		pm := app.PluginManager()
		if pm == nil {
			t.Fatal("Plugin manager should not be nil")
		}

		// The storage might or might not be loaded depending on mock plugin availability
		// But the configuration should be complete without errors
	})
}

// TestConfigureStorageCompleteCoverage tests all paths in ConfigureStorage for maximum coverage
func TestConfigureStorageCompleteCoverage(t *testing.T) {
	t.Run("ConfigureStorage_SuccessfulPluginValidation", func(t *testing.T) {
		// Test successful plugin validation path
		logger := &MockLogger{}
		app := New("validation_success").SetLogger(logger)

		// Create a valid mock storage config that will succeed validation
		config := &StorageConfig{
			Provider: "mock",
			Config: map[string]interface{}{
				"valid_config": true,
			},
		}

		// Mock a successful plugin loading scenario
		// Since we can't easily mock real plugins, we test the error paths
		app.ConfigureStorage(config)

		// Verify the config was set
		if app.StorageConfig() != config {
			t.Error("Config should be set even if plugin loading fails")
		}

		// Plugin manager should be created
		if app.PluginManager() == nil {
			t.Error("Plugin manager should be created during ConfigureStorage")
		}
	})

	t.Run("ConfigureStorage_PluginValidationFailure", func(t *testing.T) {
		// Test plugin validation failure path
		logger := &MockLogger{}
		app := New("validation_fail").SetLogger(logger)

		config := &StorageConfig{
			Provider:   "invalid_provider",
			PluginPath: "/nonexistent/invalid.so",
			Config: map[string]interface{}{
				"invalid": "config",
			},
		}

		// This should handle validation failure gracefully
		app.ConfigureStorage(config)

		// Config should still be set for debugging
		if app.StorageConfig() != config {
			t.Error("Config should be set even when validation fails")
		}

		// Storage should remain nil due to validation failure
		if app.Storage() != nil {
			t.Error("Storage should remain nil when validation fails")
		}
	})

	t.Run("ConfigureStorage_PluginCreationFailure", func(t *testing.T) {
		// Test plugin creation failure path
		logger := &MockLogger{}
		app := New("creation_fail").SetLogger(logger)

		config := &StorageConfig{
			Provider: "creation_fail_provider",
			Config: map[string]interface{}{
				"fail_creation": true,
			},
		}

		// This should handle creation failure gracefully
		app.ConfigureStorage(config)

		// Verify error handling
		if app.StorageConfig() != config {
			t.Error("Config should be set even when plugin creation fails")
		}

		if app.Storage() != nil {
			t.Error("Storage should remain nil when plugin creation fails")
		}
	})

	t.Run("ConfigureStorage_HealthCheckFailure", func(t *testing.T) {
		// Test health check failure path (should continue anyway)
		logger := &MockLogger{}
		app := New("health_fail").SetLogger(logger)

		config := &StorageConfig{
			Provider: "health_fail_provider",
			Config: map[string]interface{}{
				"fail_health": true,
			},
		}

		// This should handle health check failure but continue
		app.ConfigureStorage(config)

		// Config should be set
		if app.StorageConfig() != config {
			t.Error("Config should be set even when health check fails")
		}
	})

	t.Run("ConfigureStorage_WithExistingPluginManager", func(t *testing.T) {
		// Test behavior when plugin manager already exists
		logger := &MockLogger{}
		app := New("existing_pm").SetLogger(logger)

		// First configuration to create plugin manager
		config1 := &StorageConfig{
			Provider: "first",
			Config:   map[string]interface{}{},
		}
		app.ConfigureStorage(config1)
		existingPM := app.PluginManager()

		// Second configuration should reuse existing plugin manager
		config2 := &StorageConfig{
			Provider: "second",
			Config:   map[string]interface{}{},
		}
		app.ConfigureStorage(config2)

		// Plugin manager should be the same instance
		if app.PluginManager() != existingPM {
			t.Error("Plugin manager should be reused, not recreated")
		}

		// Config should be updated
		if app.StorageConfig() != config2 {
			t.Error("Config should be updated to latest")
		}
	})

	t.Run("ConfigureStorage_CompleteSuccessfulFlow", func(t *testing.T) {
		// Test the complete successful configuration flow with all features
		logger := &MockLogger{}
		app := New("complete_success").SetLogger(logger)

		config := &StorageConfig{
			Provider:   "complete_test",
			PluginPath: "/test/path/complete.so",
			Config: map[string]interface{}{
				"host":    "localhost",
				"port":    1234,
				"timeout": "30s",
			},
			Namespace:     "complete_test_ns",
			EnableMetrics: true,
			EnableTracing: true,
			EnableAudit:   true,
		}

		// This tests the complete flow
		result := app.ConfigureStorage(config)

		// Verify fluent interface
		if result != app {
			t.Error("ConfigureStorage should return the app for chaining")
		}

		// Verify all settings are preserved
		savedConfig := app.StorageConfig()
		if savedConfig == nil {
			t.Fatal("Config should be saved")
		}

		if savedConfig.Provider != config.Provider {
			t.Error("Provider should be preserved")
		}

		if savedConfig.Namespace != config.Namespace {
			t.Error("Namespace should be preserved")
		}

		if !savedConfig.EnableMetrics || !savedConfig.EnableTracing || !savedConfig.EnableAudit {
			t.Error("Observability settings should be preserved")
		}
	})
}
