// Security Test Suite for Plugin Discovery System
// Red Team / Adversarial Testing Approach
//
// This test suite takes an adversarial approach to testing the plugin discovery
// system, simulating real-world attack scenarios that malicious actors might
// attempt against the Orpheus storage plugin system.
//
// Test Categories:
// 1. Path Traversal & Directory Manipulation Attacks
// 2. Symlink & Filesystem Race Condition Attacks
// 3. Malicious Plugin File Attacks
// 4. Memory Exhaustion & Resource Depletion Attacks
// 5. Concurrent Access & Race Condition Attacks
// 6. Symbol Resolution & Type Confusion Attacks
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// SecurityTestHarness provides utilities for adversarial security testing
type SecurityTestHarness struct {
	tempDir    string
	testLogger *MockLogger
	cleanup    []func()
}

func newSecurityTestHarness(t *testing.T) *SecurityTestHarness {
	tempDir := createTempDir(t)
	return &SecurityTestHarness{
		tempDir:    tempDir,
		testLogger: &MockLogger{},
		cleanup:    []func(){},
	}
}

func (h *SecurityTestHarness) teardown(t *testing.T) {
	for i := len(h.cleanup) - 1; i >= 0; i-- {
		h.cleanup[i]()
	}
	if err := os.RemoveAll(h.tempDir); err != nil {
		t.Logf("Warning: failed to remove temp dir: %v", err)
	}
}

// PATH TRAVERSAL & DIRECTORY MANIPULATION ATTACKS

func TestSecurityPathTraversalAttacks(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        1024,
		AllowedPaths:         []string{harness.tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}
	pm := NewPluginManager(harness.testLogger, config)

	t.Run("DoublePathTraversal", func(t *testing.T) {
		// Attempt to escape allowed directory using ../../../
		maliciousPath := filepath.Join(harness.tempDir, "..", "..", "..", "etc", "passwd")

		_, err := pm.LoadPlugin(context.Background(), maliciousPath)
		if err == nil {
			t.Error("Expected error for path traversal attack, but plugin loaded successfully")
		}

		if !strings.Contains(err.Error(), "plugin path not in allowed paths") {
			t.Errorf("Expected path validation error, got: %v", err)
		}
	})

	t.Run("NullByteInjection", func(t *testing.T) {
		// Attempt null byte injection to bypass path checks
		maliciousPath := harness.tempDir + "/safe.so\x00../../../etc/passwd"

		_, err := pm.LoadPlugin(context.Background(), maliciousPath)
		if err == nil {
			t.Error("Expected error for null byte injection, but plugin loaded")
		}
	})

	t.Run("UnicodeNormalizationAttack", func(t *testing.T) {
		// Use Unicode normalization to bypass path checks
		// Using composed vs decomposed Unicode characters
		maliciousPath := harness.tempDir + "/test\u0065\u0301.so" // é using combining chars

		_, err := pm.LoadPlugin(context.Background(), maliciousPath)
		// Should fail because file doesn't exist, not because of path validation
		if err != nil && !strings.Contains(err.Error(), "plugin file not accessible") {
			t.Logf("Unicode normalization handled correctly: %v", err)
		}
	})

	t.Run("WindowsDeviceNameAttack", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-specific attack, skipping on non-Windows")
		}

		// Attempt to access Windows device files
		devicePaths := []string{"CON", "PRN", "AUX", "NUL", "COM1", "LPT1"}
		for _, device := range devicePaths {
			maliciousPath := filepath.Join(harness.tempDir, device+".so")
			_, err := pm.LoadPlugin(context.Background(), maliciousPath)
			if err == nil {
				t.Errorf("Expected error for device name attack with %s", device)
			}
		}
	})

	t.Run("ExcessiveDirectoryDepth", func(t *testing.T) {
		// Create deeply nested directory structure to test recursion limits
		deepPath := harness.tempDir
		for i := 0; i < 1000; i++ {
			deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
		}
		deepPath = filepath.Join(deepPath, "plugin.so")

		_, err := pm.LoadPlugin(context.Background(), deepPath)
		if err == nil {
			t.Error("Expected error for excessively deep path")
		}
	})
}

// SYMLINK & FILESYSTEM RACE CONDITION ATTACKS

func TestSecuritySymlinkAttacks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink tests require Unix-like OS")
	}

	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        1024,
		AllowedPaths:         []string{harness.tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}
	pm := NewPluginManager(harness.testLogger, config)

	t.Run("SymlinkToForbiddenPath", func(t *testing.T) {
		// Create a symlink pointing outside allowed directory
		symlinkPath := filepath.Join(harness.tempDir, "evil.so")
		targetPath := "/etc/passwd"

		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			t.Skipf("Cannot create symlink: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), symlinkPath)
		if err == nil {
			t.Error("Expected error for symlink pointing outside allowed paths")
		}
	})

	t.Run("SymlinkLoop", func(t *testing.T) {
		// Create circular symlinks to cause infinite loop
		link1 := filepath.Join(harness.tempDir, "link1.so")
		link2 := filepath.Join(harness.tempDir, "link2.so")

		if err := os.Symlink(link2, link1); err != nil {
			t.Skipf("Cannot create symlink: %v", err)
		}
		if err := os.Symlink(link1, link2); err != nil {
			t.Skipf("Cannot create symlink: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), link1)
		if err == nil {
			t.Error("Expected error for symlink loop")
		}
	})

	t.Run("TOCTOUFileReplacement", func(t *testing.T) {
		// Time-of-Check-Time-of-Use attack: replace file after validation
		pluginPath := filepath.Join(harness.tempDir, "toctou.so")

		// Create legitimate file first
		if err := os.WriteFile(pluginPath, []byte("legitimate content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Start plugin loading in goroutine
		var loadErr error
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			// Add small delay to increase chance of race condition
			time.Sleep(1 * time.Millisecond)
			_, loadErr = pm.LoadPlugin(context.Background(), pluginPath)
		}()

		// Immediately replace with malicious content
		maliciousContent := []byte("malicious content that should not be loaded")
		if err := os.WriteFile(pluginPath, maliciousContent, 0644); err != nil {
			t.Logf("Warning: failed to replace file for TOCTOU test: %v", err)
		}

		wg.Wait()

		// Should fail because it's not a valid plugin
		if loadErr == nil {
			t.Error("Expected error for TOCTOU attack, but plugin loaded")
		}
	})
}

// MALICIOUS PLUGIN FILE ATTACKS

func TestSecurityMaliciousPluginFiles(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        1024 * 1024, // 1MB for this test
		AllowedPaths:         []string{harness.tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}
	pm := NewPluginManager(harness.testLogger, config)

	t.Run("MalformedELFHeader", func(t *testing.T) {
		// Create file with malformed ELF header to crash plugin.Open
		malformedPath := filepath.Join(harness.tempDir, "malformed.so")
		malformedELF := []byte{0x7f, 0x45, 0x4c, 0x46, 0xFF, 0xFF, 0xFF, 0xFF} // Bad ELF header

		if err := os.WriteFile(malformedPath, malformedELF, 0644); err != nil {
			t.Fatalf("Failed to create malformed file: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), malformedPath)
		if err == nil {
			t.Error("Expected error for malformed ELF file")
		}

		if !strings.Contains(err.Error(), "failed to open plugin") {
			t.Errorf("Expected plugin opening error, got: %v", err)
		}
	})

	t.Run("BinaryWithNullBytes", func(t *testing.T) {
		// Create binary file with embedded null bytes
		nullBytesPath := filepath.Join(harness.tempDir, "nulls.so")
		nullContent := make([]byte, 1024)
		// Fill with null bytes and some random data
		for i := 0; i < len(nullContent); i += 2 {
			nullContent[i] = 0x00
			if i+1 < len(nullContent) {
				nullContent[i+1] = byte(i % 256)
			}
		}

		if err := os.WriteFile(nullBytesPath, nullContent, 0644); err != nil {
			t.Fatalf("Failed to create null bytes file: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), nullBytesPath)
		if err == nil {
			t.Error("Expected error for null bytes file")
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Test loading completely empty file
		emptyPath := filepath.Join(harness.tempDir, "empty.so")

		if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), emptyPath)
		if err == nil {
			t.Error("Expected error for empty file")
		}
	})

	t.Run("FileWithExecutableBit", func(t *testing.T) {
		// Test file with executable permissions (potential security risk)
		execPath := filepath.Join(harness.tempDir, "executable.so")

		if err := os.WriteFile(execPath, []byte("fake plugin"), 0755); err != nil {
			t.Fatalf("Failed to create executable file: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), execPath)
		// Should still fail due to invalid plugin format, but should handle gracefully
		if err != nil && !strings.Contains(err.Error(), "failed to open plugin") {
			t.Errorf("Unexpected error type for executable file: %v", err)
		}
	})
}

// MEMORY EXHAUSTION & RESOURCE DEPLETION ATTACKS

func TestSecurityResourceExhaustionAttacks(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	t.Run("ExcessivePluginSize", func(t *testing.T) {
		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024, // Small limit
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Create file exceeding size limit
		largePath := filepath.Join(harness.tempDir, "large.so")
		largeContent := make([]byte, 2048) // Exceeds 1024 limit

		if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), largePath)
		if err == nil {
			t.Error("Expected error for file exceeding size limit")
		}

		if !strings.Contains(err.Error(), "plugin file too large") {
			t.Errorf("Expected size limit error, got: %v", err)
		}
	})

	t.Run("MemoryExhaustionViaDiscovery", func(t *testing.T) {
		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024 * 1024, // 1MB
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Create thousands of fake plugin files
		const numFiles = 10000
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("plugin%d.so", i)
			filePath := filepath.Join(harness.tempDir, filename)
			if err := os.WriteFile(filePath, []byte("fake"), 0644); err != nil {
				t.Fatalf("Failed to create test file %d: %v", i, err)
			}
		}

		// Attempt discovery - should handle gracefully without excessive memory usage
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		plugins, err := pm.DiscoverPlugins(ctx)
		if err != nil {
			t.Logf("Discovery failed as expected: %v", err)
		}

		if len(plugins) > 0 {
			t.Logf("Discovered %d plugins (expected for stress test)", len(plugins))
		}

		// Check if process is still responsive
		runtime.GC()
	})

	t.Run("InfiniteRecursionViaDirectories", func(t *testing.T) {
		// Create deeply nested directory structure
		deepPath := harness.tempDir
		for i := 0; i < 50; i++ {
			deepPath = filepath.Join(deepPath, fmt.Sprintf("deep%d", i))
			if err := os.MkdirAll(deepPath, 0755); err != nil {
				t.Fatalf("Failed to create deep directory: %v", err)
			}
		}

		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Discovery should complete without stack overflow
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := pm.DiscoverPlugins(ctx)
		// Should complete within timeout
		if err != nil {
			t.Logf("Discovery handled deep directories: %v", err)
		}
	})
}

// CONCURRENT ACCESS & RACE CONDITION ATTACKS

func TestSecurityConcurrencyAttacks(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	config := &PluginSecurityConfig{
		AllowUnsignedPlugins: true,
		ValidateChecksums:    false,
		MaxPluginSize:        1024,
		AllowedPaths:         []string{harness.tempDir},
		RequiredSymbols:      []string{"NewStoragePlugin"},
	}

	t.Run("ConcurrentRegistryCorruption", func(t *testing.T) {
		pm := NewPluginManager(harness.testLogger, config)

		const numGoroutines = 100
		var wg sync.WaitGroup

		// Attempt to corrupt registry through concurrent access
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Mix of operations that should be thread-safe
				pluginName := fmt.Sprintf("plugin%d", id%10)

				// Try to get non-existent plugin
				_, _ = pm.GetPlugin(pluginName)

				// List loaded plugins
				_ = pm.ListLoadedPlugins()

				// Try to unload non-existent plugin
				_ = pm.UnloadPlugin(context.Background(), pluginName)

				// Try plugin discovery
				_, _ = pm.DiscoverPlugins(context.Background())
			}(i)
		}

		wg.Wait()

		// Registry should still be functional
		plugins := pm.ListLoadedPlugins()
		if plugins == nil {
			t.Error("Registry became nil after concurrent access")
		}
	})

	t.Run("DeadlockAttack", func(t *testing.T) {
		pm := NewPluginManager(harness.testLogger, config)

		// Create channels to coordinate goroutines
		ready := make(chan bool, 2)
		done := make(chan bool, 2)

		// Goroutine 1: Long-running discovery
		go func() {
			ready <- true
			for i := 0; i < 1000; i++ {
				_, _ = pm.DiscoverPlugins(context.Background())
				if i%100 == 0 {
					runtime.Gosched() // Yield to other goroutines
				}
			}
			done <- true
		}()

		// Goroutine 2: Rapid registry access
		go func() {
			ready <- true
			for i := 0; i < 1000; i++ {
				_ = pm.ListLoadedPlugins()
				_, _ = pm.GetPlugin(fmt.Sprintf("plugin%d", i%10))
				if i%100 == 0 {
					runtime.Gosched()
				}
			}
			done <- true
		}()

		// Wait for goroutines to be ready
		<-ready
		<-ready

		// Wait for completion with timeout
		timeout := time.After(10 * time.Second)
		completed := 0

		for completed < 2 {
			select {
			case <-done:
				completed++
			case <-timeout:
				t.Error("Potential deadlock detected - operations didn't complete within timeout")
				return
			}
		}

		t.Log("Concurrent operations completed successfully")
	})
}

// SYMBOL RESOLUTION & TYPE CONFUSION ATTACKS

func TestSecuritySymbolAttacks(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	t.Run("MissingRequiredSymbols", func(t *testing.T) {
		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin", "MissingSymbol", "AnotherMissing"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Create fake plugin file
		fakePath := filepath.Join(harness.tempDir, "fake.so")
		if err := os.WriteFile(fakePath, []byte("fake plugin"), 0644); err != nil {
			t.Fatalf("Failed to create fake plugin: %v", err)
		}

		_, err := pm.LoadPlugin(context.Background(), fakePath)
		if err == nil {
			t.Error("Expected error for missing required symbols")
		}
	})

	t.Run("ExcessiveSymbolRequirements", func(t *testing.T) {
		// Test with hundreds of required symbols to check performance impact
		var requiredSymbols []string
		for i := 0; i < 1000; i++ {
			requiredSymbols = append(requiredSymbols, fmt.Sprintf("Symbol%d", i))
		}

		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      requiredSymbols,
		}
		pm := NewPluginManager(harness.testLogger, config)

		fakePath := filepath.Join(harness.tempDir, "fake.so")
		if err := os.WriteFile(fakePath, []byte("fake plugin"), 0644); err != nil {
			t.Fatalf("Failed to create fake plugin: %v", err)
		}

		start := time.Now()
		_, err := pm.LoadPlugin(context.Background(), fakePath)
		duration := time.Since(start)

		if err == nil {
			t.Error("Expected error for plugin without required symbols")
		}

		if duration > 5*time.Second {
			t.Errorf("Symbol validation took too long: %v", duration)
		}
	})
}

// EDGE CASE SECURITY TESTS

func TestSecurityAdvancedEdgeCases(t *testing.T) {
	harness := newSecurityTestHarness(t)
	defer harness.teardown(t)

	t.Run("NilConfigurationAttack", func(t *testing.T) {
		// Test with nil configuration - should use secure defaults
		pm := NewPluginManager(harness.testLogger, nil)

		if pm.securityConfig == nil {
			t.Error("Plugin manager should have default security config when nil is provided")
		}

		if pm.securityConfig.AllowUnsignedPlugins {
			t.Error("Default config should not allow unsigned plugins")
		}

		if !pm.securityConfig.ValidateChecksums {
			t.Error("Default config should validate checksums")
		}
	})

	t.Run("ModifiedConfigAfterCreation", func(t *testing.T) {
		config := DefaultPluginSecurityConfig()
		originalAllowUnsigned := config.AllowUnsignedPlugins
		pm := NewPluginManager(harness.testLogger, config)

		// Attacker tries to modify config after plugin manager creation
		config.AllowUnsignedPlugins = true
		config.AllowedPaths = []string{"/"}
		config.MaxPluginSize = 1024 * 1024 * 1024 // 1GB

		// Plugin manager should not be affected by external config changes (defensive copy)
		if pm.securityConfig.AllowUnsignedPlugins != originalAllowUnsigned {
			t.Error("Plugin manager security config was externally modified - defensive copy failed")
		}

		// Verify all security settings remain unchanged
		if len(pm.securityConfig.AllowedPaths) == 1 && pm.securityConfig.AllowedPaths[0] == "/" {
			t.Error("Allowed paths were externally modified - defensive copy failed")
		}

		if pm.securityConfig.MaxPluginSize == 1024*1024*1024 {
			t.Error("Max plugin size was externally modified - defensive copy failed")
		}
	})

	t.Run("MemoryCorruptionAttempt", func(t *testing.T) {
		pm := NewPluginManager(harness.testLogger, DefaultPluginSecurityConfig())

		// Attempt to corrupt internal state using unsafe operations
		// This is a demonstration - real attacks would be more sophisticated
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from memory corruption attempt: %v", r)
			}
		}()

		// Try to access internal registry unsafely
		registryPtr := unsafe.Pointer(&pm.registry)
		if registryPtr == nil {
			t.Log("Cannot perform unsafe memory operations in this environment")
		}

		// Plugin manager should remain functional
		plugins := pm.ListLoadedPlugins()
		if plugins == nil {
			t.Error("Registry became corrupted")
		}
	})

	t.Run("ExcessivePathLengths", func(t *testing.T) {
		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Create extremely long path that exceeds filesystem limits
		longPath := harness.tempDir
		for len(longPath) < 4096 { // Most filesystems have path limits around 4096
			longPath = filepath.Join(longPath, "verylongdirectoryname")
		}
		longPath = filepath.Join(longPath, "plugin.so")

		_, err := pm.LoadPlugin(context.Background(), longPath)
		if err == nil {
			t.Error("Expected error for excessively long path")
		}

		// Should handle gracefully without crashing
		t.Logf("Long path handled correctly: %v", err)
	})

	t.Run("SpecialFilesystemEntries", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-specific filesystem test")
		}

		config := &PluginSecurityConfig{
			AllowUnsignedPlugins: true,
			ValidateChecksums:    false,
			MaxPluginSize:        1024,
			AllowedPaths:         []string{harness.tempDir},
			RequiredSymbols:      []string{"NewStoragePlugin"},
		}
		pm := NewPluginManager(harness.testLogger, config)

		// Test various special filesystem entries
		specialPaths := []string{
			"/dev/null",
			"/dev/zero",
			"/dev/random",
			"/proc/self/mem",
		}

		for _, specialPath := range specialPaths {
			_, err := pm.LoadPlugin(context.Background(), specialPath)
			if err == nil {
				t.Errorf("Expected error for special file: %s", specialPath)
			}
		}
	})
}

// FUZZING SUPPORT FOR SECURITY TESTS

// FuzzSecurityPluginPaths performs fuzz testing on plugin path validation
func FuzzSecurityPluginPaths(f *testing.F) {
	// Seed with known attack vectors
	f.Add("../../../etc/passwd")
	f.Add("../../../../../../../../etc/passwd")
	f.Add("/dev/null")
	f.Add("/dev/zero")
	f.Add("CON.so")
	f.Add("test\x00.so")
	f.Add("test\u0065\u0301.so")
	f.Add(strings.Repeat("A", 1000))
	f.Add("./test/../../../../../../../etc/passwd")
	f.Add("\\\\server\\share\\plugin.so")
	f.Add("/proc/self/mem")
	f.Add("plugin.so\x00/etc/passwd")

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

	f.Fuzz(func(t *testing.T, pluginPath string) {
		pm := NewPluginManager(&MockLogger{}, config)

		// Should never panic, regardless of input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic on plugin path %q: %v", pluginPath, r)
			}
		}()

		_, err := pm.LoadPlugin(context.Background(), pluginPath)

		// Most fuzzed inputs should result in errors
		if err == nil {
			t.Logf("Unexpectedly succeeded with path: %q", pluginPath)
		}

		// Ensure error messages don't leak sensitive information
		if err != nil {
			errorMsg := err.Error()
			sensitiveTerms := []string{
				"/etc/passwd",
				"/etc/shadow",
				"password",
				"secret",
				"private",
			}

			for _, term := range sensitiveTerms {
				if strings.Contains(strings.ToLower(errorMsg), term) {
					t.Logf("Potential information leak in error for path %q: %s", pluginPath, errorMsg)
				}
			}
		}
	})
}

// Benchmark security-related operations to ensure they don't degrade performance
func BenchmarkSecurityValidation(b *testing.B) {
	tempDir := createTempDir(b)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	config := DefaultPluginSecurityConfig()
	config.AllowedPaths = []string{tempDir}
	pm := NewPluginManager(&MockLogger{}, config)

	// Create a test file
	testPath := filepath.Join(tempDir, "test.so")
	if err := os.WriteFile(testPath, []byte("fake plugin content"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.Run("PathValidation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pm.validatePluginPath(testPath)
		}
	})

	b.Run("FileValidation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pm.validatePluginFile(testPath)
		}
	})

	b.Run("HashCalculation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = pm.calculateFileHash(testPath)
		}
	})
}
