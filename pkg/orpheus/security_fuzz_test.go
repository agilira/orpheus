// Advanced Security Fuzzing for Plugin System
// Specialized Fuzzer for Path Manipulation and Plugin Content Attacks
//
// This fuzzer specifically targets the plugin discovery system with
// adversarial inputs designed to find edge cases and security vulnerabilities.
// Focuses on:
// - Path manipulation attacks (traversal, injection, encoding)
// - Binary content fuzzing for malformed plugins
// - Concurrent attack scenarios
// - Resource exhaustion patterns
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzPluginPathSecurity performs comprehensive path manipulation fuzzing
func FuzzPluginPathSecurity(f *testing.F) {
	// Path traversal seeds
	f.Add("../../../etc/passwd")
	f.Add("..\\..\\..\\windows\\system32\\config\\sam")
	f.Add("../../../../../../../../../../../../etc/passwd")
	f.Add("/../../etc/passwd")
	f.Add("\\..\\..\\etc\\passwd")

	// Null byte injection
	f.Add("safe.so\x00../../etc/passwd")
	f.Add("test\x00.so")
	f.Add("/tmp/safe\x00/../../etc/passwd")

	// URL encoding attacks
	f.Add("..%2F..%2F..%2Fetc%2Fpasswd")
	f.Add("..%252F..%252F..%252Fetc%252Fpasswd")     // Double encoding
	f.Add("..%c0%af..%c0%af..%c0%afetc%c0%afpasswd") // UTF-8 overlong

	// Unicode normalization
	f.Add("..%E2%80%8F/..%E2%80%8F/etc/passwd") // Right-to-left override
	f.Add("test\u0065\u0301.so")                // Composed vs decomposed
	f.Add("test\u202e.so")                      // Right-to-left override

	// Windows device names
	f.Add("CON.so")
	f.Add("PRN.so")
	f.Add("AUX.so")
	f.Add("NUL.so")
	f.Add("COM1.so")
	f.Add("LPT1.so")
	f.Add("con.so")  // Case variations
	f.Add("CON.so:") // With stream

	// Long paths
	f.Add(strings.Repeat("A", 1000) + ".so")
	f.Add(strings.Repeat("../", 500) + "etc/passwd")

	// Special characters and encoding
	f.Add("test\r\n.so")
	f.Add("test\t.so")
	f.Add("test\b.so")
	f.Add("test\f.so")
	f.Add("test\v.so")
	f.Add("test;rm -rf /.so")
	f.Add("test`rm -rf /`.so")
	f.Add("test$(rm -rf /).so")
	f.Add("test|rm -rf |.so")
	f.Add("test&rm -rf &.so")

	// Network paths
	f.Add("\\\\malicious-server\\share\\plugin.so")
	f.Add("file://malicious-server/share/plugin.so")
	f.Add("smb://malicious-server/share/plugin.so")

	// Proc filesystem attacks (Unix)
	f.Add("/proc/self/mem")
	f.Add("/proc/self/maps")
	f.Add("/proc/version")
	f.Add("/proc/self/environ")

	// Symbolic paths
	f.Add("./plugin.so")
	f.Add("../plugin.so")
	f.Add("~/plugin.so")
	f.Add("~root/plugin.so")
	f.Add("~/../../../etc/passwd")

	// Setup test harness
	tempDir := createTempDir(f)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			f.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        10240, // 10KB
		AllowedPaths:         []string{tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}

	f.Fuzz(func(t *testing.T, pluginPath string) {
		pm := NewPluginManager(&MockLogger{}, config)

		// Track potential security violations
		var violations []string

		// Ensure no panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SECURITY VIOLATION: Panic with path %q: %v", pluginPath, r)
			}
		}()

		// Test path normalization safety
		if strings.Contains(pluginPath, "..") && !strings.Contains(pluginPath, tempDir) {
			if abs, err := filepath.Abs(pluginPath); err == nil {
				if !strings.HasPrefix(abs, tempDir) {
					violations = append(violations, "potential path traversal")
				}
			}
		}

		// Check for dangerous patterns
		dangerousPatterns := []string{
			"/etc/passwd", "/etc/shadow", "/etc/hosts",
			"windows\\system32", "system.ini", "boot.ini",
			"/proc/", "/sys/", "/dev/",
			"CON", "PRN", "AUX", "NUL", "COM", "LPT",
		}

		upperPath := strings.ToUpper(pluginPath)
		for _, pattern := range dangerousPatterns {
			if strings.Contains(upperPath, strings.ToUpper(pattern)) {
				violations = append(violations, fmt.Sprintf("dangerous pattern: %s", pattern))
			}
		}

		// Check for encoding attacks
		if !utf8.ValidString(pluginPath) {
			violations = append(violations, "invalid UTF-8 encoding")
		}

		if strings.ContainsAny(pluginPath, "\x00\r\n\t") {
			violations = append(violations, "control characters detected")
		}

		// Test actual plugin loading
		start := time.Now()
		_, err := pm.LoadPlugin(context.Background(), pluginPath)
		duration := time.Since(start)

		// Should complete quickly even with malicious input
		if duration > 1*time.Second {
			t.Errorf("PERFORMANCE VIOLATION: Path processing took %v for: %q", duration, pluginPath)
		}

		// Should reject most fuzzed inputs
		if err == nil && len(violations) > 0 {
			t.Errorf("SECURITY VIOLATION: Accepted dangerous path %q with violations: %v",
				pluginPath, violations)
		}

		// Check error message safety (no information leakage)
		if err != nil {
			errorMsg := strings.ToLower(err.Error())
			leakageTerms := []string{
				"password", "secret", "private", "token", "key",
				"/etc/", "/proc/", "system32", "config",
			}

			for _, term := range leakageTerms {
				if strings.Contains(errorMsg, term) && !strings.Contains(pluginPath, term) {
					t.Logf("POTENTIAL INFO LEAK: Error contains %q for path %q: %s",
						term, pluginPath, err.Error())
				}
			}
		}

		// Verify plugin manager state remains valid after potential attack
		validatePluginManagerState(t, pm, fmt.Sprintf("FuzzPluginPathSecurity-%s", pluginPath))
	})
}

// FuzzPluginBinaryContent tests malformed plugin file content
func FuzzPluginBinaryContent(f *testing.F) {
	// ELF header variations
	f.Add([]byte{0x7f, 0x45, 0x4c, 0x46}) // Valid ELF magic
	f.Add([]byte{0x7f, 0x45, 0x4c, 0x47}) // Invalid ELF magic
	f.Add([]byte{0x00, 0x00, 0x00, 0x00}) // Null bytes
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}) // All 0xFF

	// PE header (Windows)
	f.Add([]byte{0x4d, 0x5a}) // MZ header

	// Random binary patterns
	f.Add([]byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe})
	f.Add([]byte("Not a binary file at all"))
	f.Add([]byte{}) // Empty file

	// Large headers with embedded nulls
	largeHeader := make([]byte, 1024)
	for i := range largeHeader {
		largeHeader[i] = byte(i % 256)
	}
	f.Add(largeHeader)

	// JavaScript/Shell injection attempts
	f.Add([]byte("#!/bin/sh\nrm -rf /\n"))
	f.Add([]byte("<script>alert('xss')</script>"))
	f.Add([]byte("<?php system($_GET['cmd']); ?>"))

	// Setup test environment
	tempDir := createTempDir(f)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			f.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    true,        // Enable checksum validation
		MaxPluginSize:        1024 * 1024, // 1MB
		AllowedPaths:         []string{tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}

	f.Fuzz(func(t *testing.T, content []byte) {
		pm := NewPluginManager(&MockLogger{}, config)

		// Create temporary plugin file with fuzzed content
		pluginPath := filepath.Join(tempDir, "fuzzed.so")

		// Clean up from previous iteration
		_ = os.Remove(pluginPath)

		// Write fuzzed content
		if err := os.WriteFile(pluginPath, content, 0644); err != nil {
			t.Skipf("Could not write test file: %v", err)
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SECURITY VIOLATION: Panic with content length %d: %v", len(content), r)
				t.Logf("Content hex: %s", hex.EncodeToString(content))
			}
		}()

		// Track processing time for DoS detection
		start := time.Now()
		_, err := pm.LoadPlugin(context.Background(), pluginPath)
		duration := time.Since(start)

		// Should fail quickly for invalid content
		if duration > 5*time.Second {
			t.Errorf("PERFORMANCE VIOLATION: Processing took %v for %d bytes", duration, len(content))
		}

		// Most fuzzed content should be rejected
		if err == nil {
			t.Logf("Unexpected success with content: %s", hex.EncodeToString(content[:min(32, len(content))]))
		}

		// Validate error handling
		if err != nil {
			errorMsg := err.Error()

			// Should not contain raw binary data in error messages
			if !utf8.ValidString(errorMsg) {
				t.Error("Error message contains invalid UTF-8")
			}

			// Should not be excessively long
			if len(errorMsg) > 1000 {
				t.Errorf("Error message too long: %d chars", len(errorMsg))
			}
		}

		// Verify plugin manager remains functional after processing malformed content
		validatePluginManagerState(t, pm, "FuzzPluginBinaryContent")
	})
}

// FuzzConcurrentPluginOperations tests race conditions with malicious timing
func FuzzConcurrentPluginOperations(f *testing.F) {
	// Operation type seeds
	f.Add([]byte("load"))
	f.Add([]byte("discover"))
	f.Add([]byte("get"))
	f.Add([]byte("list"))
	f.Add([]byte("unload"))
	f.Add([]byte("concurrent"))

	tempDir := createTempDir(f)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			f.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        1024,
		AllowedPaths:         []string{tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}

	f.Fuzz(func(t *testing.T, operations []byte) {
		if len(operations) == 0 {
			return
		}

		pm := NewPluginManager(&MockLogger{}, config)

		// Create a few test files
		for i := 0; i < 3; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.so", i))
			_ = os.WriteFile(testFile, []byte("fake plugin"), 0644)
		}

		// Convert fuzzed bytes to operation sequence
		maxOps := min(len(operations), 100) // Limit to prevent excessive test time

		done := make(chan bool, maxOps)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CONCURRENCY VIOLATION: Panic during concurrent operations: %v", r)
			}
		}()

		// Launch concurrent operations based on fuzzed input
		for i := 0; i < maxOps; i++ {
			opType := operations[i] % 6

			go func(op int, id int) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Goroutine %d panic: %v", id, r)
					}
					done <- true
				}()

				pluginName := fmt.Sprintf("test%d", id%3)

				switch op {
				case 0: // Load
					testFile := filepath.Join(tempDir, pluginName+".so")
					_, _ = pm.LoadPlugin(context.Background(), testFile)

				case 1: // Discover
					_, _ = pm.DiscoverPlugins(context.Background())

				case 2: // Get
					_, _ = pm.GetPlugin(pluginName)

				case 3: // List
					_ = pm.ListLoadedPlugins()

				case 4: // Unload
					_ = pm.UnloadPlugin(context.Background(), pluginName)

				case 5: // Mixed rapid operations
					_, _ = pm.GetPlugin(pluginName)
					_ = pm.ListLoadedPlugins()
					_, _ = pm.DiscoverPlugins(context.Background())
				}
			}(int(opType), i)
		}

		// Wait for completion with timeout
		timeout := time.After(10 * time.Second)
		completed := 0

		for completed < maxOps {
			select {
			case <-done:
				completed++
			case <-timeout:
				t.Errorf("DEADLOCK VIOLATION: Operations didn't complete within timeout")
				return
			}
		}

		// Verify plugin manager state is still valid after concurrent operations
		validatePluginManagerState(t, pm, "FuzzConcurrentPluginOperations")

		runtime.GC() // Force garbage collection to detect memory issues
	})
}

// FuzzPluginDiscoveryPaths tests directory traversal during discovery
func FuzzPluginDiscoveryPaths(f *testing.F) {
	// Directory structure seeds that could cause problems
	f.Add("../../../")
	f.Add("..\\..\\..\\")
	f.Add("/tmp")
	f.Add("/etc")
	f.Add("/proc")
	f.Add("/sys")
	f.Add("/dev")
	f.Add("C:\\Windows\\System32")
	f.Add("C:\\Users\\Administrator")
	f.Add(strings.Repeat("../", 100))
	f.Add(strings.Repeat("subdir/", 50))

	// Create base temp directory
	tempDir := createTempDir(f)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			f.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	f.Fuzz(func(t *testing.T, searchPath string) {
		// Ensure no panics during path traversal
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DISCOVERY PANIC with path %q: %v", searchPath, r)
			}
		}()

		// Create potentially malicious search path
		var testPaths []string
		if filepath.IsAbs(searchPath) {
			testPaths = []string{searchPath}
		} else {
			testPaths = []string{filepath.Join(tempDir, searchPath)}
		}

		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         testPaths,
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}

		pm := NewPluginManager(&MockLogger{}, config)

		// Test discovery with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := time.Now()
		plugins, err := pm.DiscoverPlugins(ctx)
		duration := time.Since(start)

		// Should complete within reasonable time
		// Allow more time for Windows and complex directory structures
		maxDuration := 5 * time.Second
		if runtime.GOOS == "windows" {
			maxDuration = 10 * time.Second
		}
		if duration > maxDuration {
			t.Errorf("PERFORMANCE VIOLATION: Discovery took %v for path %q (max allowed: %v)", duration, searchPath, maxDuration)
		}

		// Should handle errors gracefully
		if err != nil {
			errorMsg := err.Error()
			if !utf8.ValidString(errorMsg) {
				t.Error("Discovery error contains invalid UTF-8")
			}
		}

		// Results should be reasonable
		if len(plugins) > 10000 {
			t.Errorf("RESOURCE VIOLATION: Discovered suspiciously many plugins: %d", len(plugins))
		}

		// Each discovered plugin should be in allowed paths
		for _, plugin := range plugins {
			if !filepath.IsAbs(plugin) {
				t.Errorf("SECURITY VIOLATION: Non-absolute path discovered: %s", plugin)
			}

			// Should not escape to system directories (basic check)
			suspicious := []string{"/etc/", "/proc/", "/sys/", "/dev/"}
			if runtime.GOOS == "windows" {
				suspicious = []string{"C:\\Windows\\", "C:\\Program Files\\"}
			}

			for _, suspiciousPath := range suspicious {
				if strings.HasPrefix(plugin, suspiciousPath) {
					t.Logf("WARNING: Discovered plugin in system directory: %s", plugin)
				}
			}
		}
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper to validate plugin manager state after attacks
func validatePluginManagerState(t *testing.T, pm *PluginManager, testName string) {
	// Plugin manager should remain functional
	if pm == nil {
		t.Errorf("%s: Plugin manager became nil", testName)
		return
	}

	// Registry should be accessible
	plugins := pm.ListLoadedPlugins()
	if plugins == nil {
		t.Errorf("%s: Plugin registry became nil", testName)
	}

	// Should be able to perform basic operations
	_, err := pm.GetPlugin("nonexistent")
	if err == nil {
		t.Errorf("%s: GetPlugin should return error for nonexistent plugin", testName)
	}

	// Discovery should still work (even if it fails, should not crash)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, _ = pm.DiscoverPlugins(ctx)
}
