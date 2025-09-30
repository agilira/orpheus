// integration_test.go: Security Validation example integration tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// createTestApp creates a test app with the same configuration as main
func createTestApp() *orpheus.App {
	app := orpheus.New("security-validation").
		SetDescription("Demonstrate Orpheus CLI framework security validation capabilities").
		SetVersion("1.0.0")

	// Add global security flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose security logging").
		AddGlobalBoolFlag("strict", "s", false, "Enable strict security mode").
		AddGlobalBoolFlag("audit", "a", false, "Enable security audit logging")

	// Input validation command
	validateCmd := orpheus.NewCommand("validate", "Validate user input with security controls").
		AddFlag("input", "i", "", "Input string to validate (required)").
		AddFlag("type", "t", "general", "Input type (general, path, command, sql)").
		AddBoolFlag("sanitize", "s", false, "Apply input sanitization").
		AddBoolFlag("show-patterns", "p", false, "Show detected dangerous patterns").
		SetHandler(handleValidate)

	// Path scanning command
	scanCmd := orpheus.NewCommand("scan", "Scan file paths for security vulnerabilities").
		AddFlag("path", "p", "", "File path to scan (required)").
		AddFlag("operation", "o", "read", "Operation type (read, write, execute)").
		AddBoolFlag("deep-scan", "d", false, "Enable deep security scanning").
		AddBoolFlag("show-analysis", "a", false, "Show detailed security analysis").
		SetHandler(handleScan)

	// Demo command - matches main.go
	demoCmd := orpheus.NewCommand("demo", "Demonstrate security controls against attack scenarios").
		AddFlag("attack-type", "t", "", "Attack type (path-traversal, command-injection, sql-injection, xss, buffer-overflow)").
		AddBoolFlag("show-protection", "p", false, "Show protection mechanisms").
		AddBoolFlag("interactive", "i", false, "Interactive attack demonstration").
		SetHandler(handleDemo)

	// Benchmark command
	benchmarkCmd := orpheus.NewCommand("benchmark", "Benchmark security validation performance").
		AddIntFlag("iterations", "n", 1000, "Number of iterations").
		AddFlag("test-type", "t", "all", "Test type (path, input, all)").
		AddBoolFlag("detailed", "d", false, "Show detailed performance metrics").
		SetHandler(handleBenchmark)

	app.AddCommand(validateCmd)
	app.AddCommand(scanCmd)
	app.AddCommand(demoCmd)
	app.AddCommand(benchmarkCmd)

	return app
}

func TestSecurityValidationCommands(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "validate_clean_input",
			args:        []string{"validate", "--input", "clean-user-input"},
			expectError: false,
			description: "Valid user input should pass validation",
		},
		{
			name:        "validate_path_traversal_attack",
			args:        []string{"validate", "--input", "../../../etc/passwd", "--type", "path"},
			expectError: true,
			description: "Path traversal attack should be blocked",
		},
		{
			name:        "validate_command_injection",
			args:        []string{"validate", "--input", "$(rm -rf /)", "--type", "command"},
			expectError: true,
			description: "Command injection should be blocked",
		},
		{
			name:        "validate_sql_injection",
			args:        []string{"validate", "--input", "'; DROP TABLE users; --", "--type", "sql"},
			expectError: true,
			description: "SQL injection should be blocked",
		},
		{
			name:        "validate_missing_input",
			args:        []string{"validate"},
			expectError: true,
			description: "Missing input parameter should return validation error",
		},
		{
			name:        "scan_safe_path",
			args:        []string{"scan", "--path", "config.json"},
			expectError: false,
			description: "Safe file path should pass security scan",
		},
		{
			name:        "scan_system_path",
			args:        []string{"scan", "--path", "/etc/passwd"},
			expectError: true,
			description: "System file access should be blocked",
		},
		{
			name:        "scan_traversal_attack",
			args:        []string{"scan", "--path", "../../../sensitive/file"},
			expectError: true,
			description: "Path traversal in scan should be blocked",
		},

		{
			name:        "demo_path_traversal",
			args:        []string{"demo", "--attack-type", "path-traversal"},
			expectError: false,
			description: "Security demo should run and show blocked attacks",
		},
		{
			name:        "demo_command_injection",
			args:        []string{"demo", "--attack-type", "command-injection"},
			expectError: false,
			description: "Command injection demo should show security protection",
		},
		{
			name:        "demo_invalid_type",
			args:        []string{"demo", "--attack-type", "invalid-attack"},
			expectError: true,
			description: "Invalid attack type should return validation error",
		},
		{
			name:        "benchmark_all",
			args:        []string{"benchmark", "--iterations", "100", "--test-type", "all"},
			expectError: false,
			description: "Security benchmark should complete successfully",
		},
		{
			name:        "benchmark_path_only",
			args:        []string{"benchmark", "--iterations", "50", "--test-type", "path"},
			expectError: false,
			description: "Path validation benchmark should complete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s' but got none. %s", tt.name, tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v. %s", tt.name, err, tt.description)
			}

			// Additional validation for security-specific errors
			if err != nil {
				if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
					// Validate that security errors have appropriate error codes
					if orpheusErr.IsValidationError() {
						// Security validation errors should be validation errors
						if !strings.Contains(tt.name, "missing") && !strings.Contains(tt.name, "invalid") {
							// This should be a security validation failure
							if !strings.Contains(strings.ToLower(err.Error()), "security") &&
								!strings.Contains(strings.ToLower(err.Error()), "dangerous") &&
								!strings.Contains(strings.ToLower(err.Error()), "traversal") &&
								!strings.Contains(strings.ToLower(err.Error()), "injection") {
								t.Logf("Security error for '%s': %s", tt.name, err.Error())
							}
						}
					}
				}
			}
		})
	}
}

func TestSecurityValidationHelp(t *testing.T) {
	app := createTestApp()

	// Test main help
	err := app.Run([]string{"--help"})
	if err != nil {
		t.Errorf("Help command should not return error: %v", err)
	}

	// Test command-specific help
	commands := []string{"validate", "scan", "demo", "benchmark"}
	for _, cmd := range commands {
		t.Run("help_"+cmd, func(t *testing.T) {
			err := app.Run([]string{cmd, "--help"})
			if err != nil {
				t.Errorf("Help for command '%s' should not return error: %v", cmd, err)
			}
		})
	}
}

func TestSecurityValidationVersion(t *testing.T) {
	app := createTestApp()

	err := app.Run([]string{"--version"})
	if err != nil {
		t.Errorf("Version command should not return error: %v", err)
	}
}

func TestSecurityValidationErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "unknown_command",
			args:        []string{"unknown-command"},
			description: "Unknown command should return appropriate error",
		},
		{
			name:        "validate_without_input",
			args:        []string{"validate"},
			description: "Validate command without input should return validation error",
		},
		{
			name:        "scan_without_path",
			args:        []string{"scan"},
			description: "Scan command without path should return validation error",
		},

		{
			name:        "demo_without_attack_type",
			args:        []string{"demo"},
			description: "Demo command without attack type should return validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := createTestApp()
			err := app.Run(tt.args)

			if err == nil {
				t.Errorf("Expected error for test '%s' but got none. %s", tt.name, tt.description)
				return
			}

			// Validate error type and content
			if orpheusErr, ok := err.(*orpheus.OrpheusError); ok {
				if !orpheusErr.IsValidationError() && !orpheusErr.IsNotFoundError() {
					t.Errorf("Expected validation or not found error for '%s', got: %s", tt.name, orpheusErr.ErrorCode())
				}
			}
		})
	}
}
