// main_test.go: File Manager example tests
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// createTestApp creates the filemanager app for testing
func createTestApp() *orpheus.App {
	app := orpheus.New("filemanager").
		SetDescription("A simple file manager CLI demonstrating Orpheus framework").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalFlag("config", "c", "", "Configuration file path")

	// Add commands
	setupListCommand(app)
	setupSearchCommand(app)
	setupInfoCommand(app)
	setupTreeCommand(app)

	return app
}

// captureOutput captures stdout during test execution
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// createTempTestDir creates a temporary directory with test files
func createTempTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "filemanager_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test structure
	testFiles := map[string]string{
		"main.go":         "package main\n\nfunc main() {}\n",
		"config.json":     `{"name": "test"}`,
		"README.md":       "# Test Project\n",
		"test_file.txt":   "This is a test file\n",
		"subdir/test.go":  "package test\n",
		"subdir/data.txt": "Some data\n",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		dir := filepath.Dir(fullPath)

		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}

		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}

	}

	return tempDir
}

func TestAppCreation(t *testing.T) {
	app := createTestApp()

	if app.Name() != "filemanager" {
		t.Errorf("Expected app name 'filemanager', got %s", app.Name())
	}

	if app.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", app.Version())
	}

	commands := app.GetCommands()
	expectedCommands := []string{"list", "search", "info", "tree"}

	for _, cmd := range expectedCommands {
		if _, exists := commands[cmd]; !exists {
			t.Errorf("Expected command '%s' to exist", cmd)
		}
	}
}

func TestListCommand(t *testing.T) {
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "basic list",
			args:     []string{"list", "--path", tempDir},
			contains: []string{"main.go", "config.json", "README.md", "test_file.txt", "subdir/"},
		},
		{
			name:     "long format",
			args:     []string{"list", "--path", tempDir, "--long"},
			contains: []string{"-rw-", "main.go", "config.json"},
		},
		{
			name:     "with limit",
			args:     []string{"list", "--path", tempDir, "--limit", "2"},
			contains: []string{}, // We'll count the lines separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := app.Run(tt.args)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}

			// Special test for limit
			if tt.name == "with limit" {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) > 2 {
					t.Errorf("Expected at most 2 lines with limit=2, got %d lines", len(lines))
				}
			}
		})
	}
}

func TestSearchCommand(t *testing.T) {
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	app := createTestApp()

	tests := []struct {
		name        string
		args        []string
		contains    []string
		notContains []string
	}{
		{
			name:        "search go files",
			args:        []string{"search", "--pattern", "*.go", "--dir", tempDir},
			contains:    []string{"main.go", "test.go"},
			notContains: []string{".json", ".md", ".txt"},
		},
		{
			name:        "search with extension filter",
			args:        []string{"search", "--pattern", "*", "--dir", tempDir, "--ext", "txt"},
			contains:    []string{"test_file.txt", "data.txt"},
			notContains: []string{".go", ".json", ".md"},
		},
		{
			name:        "search all files",
			args:        []string{"search", "--pattern", "*", "--dir", tempDir},
			contains:    []string{"test_file.txt", "data.txt"},
			notContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := app.Run(tt.args)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain '%s', got: %s", notExpected, output)
				}
			}
		})
	}
}

func TestInfoCommand(t *testing.T) {
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	app := createTestApp()
	testFile := filepath.Join(tempDir, "main.go")

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "basic info",
			args:     []string{"info", testFile},
			contains: []string{"File:", "main.go", "Type:", "Size:", "Permissions:", "Modified:"},
		},
		{
			name:     "info with timestamps",
			args:     []string{"info", testFile, "--timestamps"},
			contains: []string{"File:", "main.go", "Modified:", "T"}, // ISO format contains T
		},
		{
			name:     "directory info",
			args:     []string{"info", tempDir},
			contains: []string{"Type: Directory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := app.Run(tt.args)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

func TestInfoCommandErrors(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorType   string
	}{
		{
			name:        "missing file argument",
			args:        []string{"info"},
			expectError: true,
			errorType:   "validation",
		},
		{
			name:        "non-existent file",
			args:        []string{"info", "/non/existent/file"},
			expectError: true,
			errorType:   "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				orpheusErr, ok := err.(*orpheus.OrpheusError)
				if !ok {
					t.Errorf("Expected OrpheusError, got %T", err)
					return
				}

				if tt.errorType == "validation" && !orpheusErr.IsValidationError() {
					t.Error("Expected validation error")
				}
				if tt.errorType == "not_found" && !orpheusErr.IsNotFoundError() {
					t.Error("Expected not found error")
				}
			} else if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestTreeCommand(t *testing.T) {
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "basic tree",
			args:     []string{"tree", "--root", tempDir},
			contains: []string{"├──", "└──", "main.go", "subdir/"},
		},
		{
			name:     "dirs only",
			args:     []string{"tree", "--root", tempDir, "--dirs-only"},
			contains: []string{"subdir/"},
			// Should not contain regular files when dirs-only is set
		},
		{
			name:     "limited depth",
			args:     []string{"tree", "--root", tempDir, "--depth", "2"},
			contains: []string{"├──", "subdir/"},
			// Depth 2 should show tree structure and subdirectories
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				err := app.Run(tt.args)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}

			// Special check for dirs-only
			if tt.name == "dirs only" {
				if strings.Contains(output, "main.go") {
					t.Error("dirs-only should not show regular files")
				}
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	tempDir := createTempTestDir(t)
	defer os.RemoveAll(tempDir)

	app := createTestApp()

	// Test verbose flag
	output := captureOutput(func() {
		err := app.Run([]string{"--verbose", "list", "--path", tempDir})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Listing directory:") {
		t.Error("Expected verbose output when --verbose flag is used")
	}

	// Test search with verbose
	output = captureOutput(func() {
		err := app.Run([]string{"--verbose", "search", "--pattern", "*.go", "--dir", tempDir})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Searching for pattern") {
		t.Error("Expected verbose search output when --verbose flag is used")
	}
}

func TestHelpGeneration(t *testing.T) {
	app := createTestApp()

	// Test main help
	output := captureOutput(func() {
		err := app.Run([]string{"--help"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	expectedInHelp := []string{
		"filemanager",
		"Available Commands:",
		"list", "search", "info", "tree",
		"Global Flags:",
		"--verbose",
	}

	for _, expected := range expectedInHelp {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help to contain '%s', got: %s", expected, output)
		}
	}

	// Test command-specific help
	output = captureOutput(func() {
		err := app.Run([]string{"search", "--help"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "--pattern") {
		t.Error("Expected search help to show --pattern flag")
	}
}

func TestVersionFlag(t *testing.T) {
	app := createTestApp()

	output := captureOutput(func() {
		err := app.Run([]string{"--version"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "1.0.0") {
		t.Errorf("Expected version output to contain '1.0.0', got: %s", output)
	}
}
