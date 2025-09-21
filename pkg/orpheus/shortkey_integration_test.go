// shortkey_integration_test.go: tests for short key functionality
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"testing"
)

// TestShortKeyIntegration tests that Orpheus can use the new ShortKey() method from flash-flags
func TestShortKeyIntegration(t *testing.T) {
	app := New("testapp")

	// Add global flags with short keys
	app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")
	app.AddGlobalFlag("config", "c", "config.json", "Configuration file")
	app.AddGlobalIntFlag("port", "p", 8080, "Server port")

	// Add global flag without short key
	app.AddGlobalFlag("database-url", "", "localhost", "Database URL")

	t.Run("global flags with short keys", func(t *testing.T) {
		// Test verbose flag (bool with short key)
		verboseFlag := app.globalFlags.Lookup("verbose")
		if verboseFlag == nil {
			t.Fatal("Expected to find verbose flag")
		}
		if verboseFlag.ShortKey() != "v" {
			t.Errorf("Expected short key 'v', got '%s'", verboseFlag.ShortKey())
		}
		if verboseFlag.Type() != "bool" {
			t.Errorf("Expected type 'bool', got '%s'", verboseFlag.Type())
		}

		// Test config flag (string with short key)
		configFlag := app.globalFlags.Lookup("config")
		if configFlag == nil {
			t.Fatal("Expected to find config flag")
		}
		if configFlag.ShortKey() != "c" {
			t.Errorf("Expected short key 'c', got '%s'", configFlag.ShortKey())
		}
		if configFlag.Type() != "string" {
			t.Errorf("Expected type 'string', got '%s'", configFlag.Type())
		}

		// Test port flag (int with short key)
		portFlag := app.globalFlags.Lookup("port")
		if portFlag == nil {
			t.Fatal("Expected to find port flag")
		}
		if portFlag.ShortKey() != "p" {
			t.Errorf("Expected short key 'p', got '%s'", portFlag.ShortKey())
		}
		if portFlag.Type() != "int" {
			t.Errorf("Expected type 'int', got '%s'", portFlag.Type())
		}
	})

	t.Run("global flag without short key", func(t *testing.T) {
		dbFlag := app.globalFlags.Lookup("database-url")
		if dbFlag == nil {
			t.Fatal("Expected to find database-url flag")
		}
		if dbFlag.ShortKey() != "" {
			t.Errorf("Expected empty short key, got '%s'", dbFlag.ShortKey())
		}
	})

	t.Run("isShortBooleanFlag with dynamic detection", func(t *testing.T) {
		// Test built-in flags
		if !app.isShortBooleanFlag("-h") {
			t.Error("Expected -h to be detected as boolean flag")
		}
		if !app.isShortBooleanFlag("-v") {
			t.Error("Expected -v to be detected as boolean flag")
		}

		// Test custom boolean flag should now be detected dynamically
		// Note: This tests our improved isShortBooleanFlag logic
		if !app.isShortBooleanFlag("-v") {
			t.Error("Expected custom verbose flag -v to be detected as boolean")
		}

		// Test non-boolean flags should not be detected
		if app.isShortBooleanFlag("-c") {
			t.Error("Expected -c (config string flag) to NOT be detected as boolean")
		}
		if app.isShortBooleanFlag("-p") {
			t.Error("Expected -p (port int flag) to NOT be detected as boolean")
		}
	})
}

// TestShortKeyInCommandFlags tests ShortKey() with command-specific flags
func TestShortKeyInCommandFlags(t *testing.T) {
	cmd := NewCommand("deploy", "Deploy application")

	// Add flags with short keys to command
	cmd.AddBoolFlag("force", "f", false, "Force deployment")
	cmd.AddFlag("environment", "e", "production", "Target environment")

	// Add flag without short key
	cmd.AddFlag("timeout", "", "30s", "Deployment timeout")

	t.Run("command flags with short keys", func(t *testing.T) {
		// Test force flag
		forceFlag := cmd.Flags().Lookup("force")
		if forceFlag == nil {
			t.Fatal("Expected to find force flag")
		}
		if forceFlag.ShortKey() != "f" {
			t.Errorf("Expected short key 'f', got '%s'", forceFlag.ShortKey())
		}

		// Test environment flag
		envFlag := cmd.Flags().Lookup("environment")
		if envFlag == nil {
			t.Fatal("Expected to find environment flag")
		}
		if envFlag.ShortKey() != "e" {
			t.Errorf("Expected short key 'e', got '%s'", envFlag.ShortKey())
		}
	})

	t.Run("command flag without short key", func(t *testing.T) {
		timeoutFlag := cmd.Flags().Lookup("timeout")
		if timeoutFlag == nil {
			t.Fatal("Expected to find timeout flag")
		}
		if timeoutFlag.ShortKey() != "" {
			t.Errorf("Expected empty short key, got '%s'", timeoutFlag.ShortKey())
		}
	})
}
