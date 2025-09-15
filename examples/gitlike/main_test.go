// Test suite for gitlike CLI demo
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// Global binary management for faster tests
var (
	testBinaryPath string
	buildOnce      sync.Once
	buildErr       error
)

// buildTestBinary builds the gitlike binary once for all tests
func buildTestBinary() {
	buildOnce.Do(func() {
		testBinaryPath = "./gitlike-test-shared"
		cmd := exec.Command("go", "build", "-o", testBinaryPath, ".")
		buildErr = cmd.Run()
	})
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Build binary once for all tests
	buildTestBinary()
	if buildErr != nil {
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if testBinaryPath != "" {
		os.Remove(testBinaryPath)
	}

	os.Exit(code)
}

// runGitlike runs the gitlike binary with given arguments and returns output
func runGitlike(t *testing.T, args ...string) (string, error) {
	if testBinaryPath == "" || buildErr != nil {
		t.Fatalf("Test binary not available: %v", buildErr)
	}

	cmd := exec.Command(testBinaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// setupTestEnvironment creates a clean test environment
func setupTestEnvironment(t *testing.T) func() {
	tmpDir, err := os.MkdirTemp("", "gitlike-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
	}
}

// Fast unit tests

func TestHelpCommands(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "main help",
			args:     []string{"--help"},
			contains: []string{"Usage: gitlike", "Available Commands:"},
		},
		{
			name:     "remote help",
			args:     []string{"remote", "--help"},
			contains: []string{"Usage: gitlike remote", "Available Subcommands:"},
		},
		{
			name:     "config help",
			args:     []string{"config", "--help"},
			contains: []string{"Usage: gitlike config", "Available Subcommands:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runGitlike(t, tt.args...)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

func TestBasicCommands(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Test basic command functionality in sequence for speed

	// Test config set
	output, err := runGitlike(t, "config", "set", "test.key", "test-value")
	if err != nil {
		t.Errorf("Config set failed: %v", err)
	}
	if !strings.Contains(output, "Set: test.key = test-value") {
		t.Errorf("Expected config set success message, got: %s", output)
	}

	// Test config get (persistence)
	output, err = runGitlike(t, "config", "get", "test.key")
	if err != nil {
		t.Errorf("Config get failed: %v", err)
	}
	if !strings.Contains(strings.TrimSpace(output), "test-value") {
		t.Errorf("Expected 'test-value', got: %s", strings.TrimSpace(output))
	}

	// Test remote add
	output, err = runGitlike(t, "remote", "add", "testremote", "https://github.com/test/repo.git")
	if err != nil {
		t.Errorf("Remote add failed: %v", err)
	}
	if !strings.Contains(output, "Added remote: testremote") {
		t.Errorf("Expected remote add success message, got: %s", output)
	}

	// Test remote list (persistence)
	output, err = runGitlike(t, "remote", "list")
	if err != nil {
		t.Errorf("Remote list failed: %v", err)
	}
	if !strings.Contains(output, "testremote") && !strings.Contains(output, "origin") {
		t.Errorf("Expected remotes in output, got: %s", output)
	}
}

func TestErrorHandling(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "invalid command",
			args:        []string{"invalidcommand"},
			expectError: true,
		},
		{
			name:        "config missing args",
			args:        []string{"config", "set", "onlykey"},
			expectError: true,
		},
		{
			name:        "remote missing args",
			args:        []string{"remote", "add", "onlyname"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runGitlike(t, tt.args...)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestStatusAndBranch(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Test status command
	output, err := runGitlike(t, "status")
	if err != nil {
		t.Errorf("Status command failed: %v", err)
	}
	if !strings.Contains(output, "On branch") {
		t.Errorf("Expected branch info in status, got: %s", output)
	}

	// Test branch list
	output, err = runGitlike(t, "branch", "list")
	if err != nil {
		t.Errorf("Branch list failed: %v", err)
	}
	if !strings.Contains(output, "main") && !strings.Contains(output, "develop") {
		t.Errorf("Expected branch info, got: %s", output)
	}
}

// Quick benchmark
func BenchmarkConfigOperation(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "gitlike-bench-*")
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runGitlike(&testing.T{}, "config", "set", "bench.key", "bench-value")
	}
}
