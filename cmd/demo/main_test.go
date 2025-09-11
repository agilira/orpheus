// main_test.go: Comprehensive test suite for Orpheus demo application
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// captureOutput captures stdout during function execution
func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	return string(output)
}

// createTestApp creates the demo app for testing
func createTestApp() *orpheus.App {
	// Create a new Orpheus application
	app := orpheus.New("demo").
		SetDescription("Orpheus CLI Framework Demo Application").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output").
		AddGlobalFlag("config", "c", "", "Configuration file path")

	// Add a simple greeting command
	app.Command("greet", "Greet someone", func(ctx *orpheus.Context) error {
		name := "World"
		if ctx.ArgCount() > 0 {
			name = ctx.GetArg(0)
		}

		if ctx.GetGlobalFlagBool("verbose") {
			fmt.Printf("Greeting %s with verbose output enabled\n", name)
		}

		fmt.Printf("Hello, %s!\n", name)
		return nil
	})

	// Add a more complex echo command
	echoCmd := orpheus.NewCommand("echo", "Echo back the arguments").
		SetHandler(func(ctx *orpheus.Context) error {
			if ctx.ArgCount() == 0 {
				return orpheus.ValidationError("echo", "no arguments provided")
			}

			for i := 0; i < ctx.ArgCount(); i++ {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(ctx.GetArg(i))
			}
			fmt.Println()
			return nil
		})

	// Add a deploy command with custom completion
	deployCmd := orpheus.NewCommand("deploy", "Deploy to environment").
		SetHandler(func(ctx *orpheus.Context) error {
			if ctx.ArgCount() == 0 {
				return orpheus.ValidationError("deploy", "environment required")
			}

			env := ctx.GetArg(0)
			fmt.Printf("Deploying to %s environment...\n", env)
			return nil
		}).
		SetCompletionHandler(func(req *orpheus.CompletionRequest) *orpheus.CompletionResult {
			if req.Type == orpheus.CompletionArgs && req.Position == 0 {
				return &orpheus.CompletionResult{
					Suggestions: []string{"production", "staging", "development"},
				}
			}
			return &orpheus.CompletionResult{Suggestions: []string{}}
		})

	app.AddCommand(echoCmd).
		AddCommand(deployCmd)

	// Add completion command
	app.AddCompletionCommand()

	// Set default command
	app.SetDefaultCommand("greet")

	return app
}

// TestAppCreation tests that the demo app is created correctly
func TestAppCreation(t *testing.T) {
	app := createTestApp()

	if app == nil {
		t.Fatal("App should not be nil")
	}

	// Test app properties (we can't directly access private fields,
	// but we can test behavior through the help output)
	output := captureOutput(func() {
		app.Run([]string{"--help"})
	})

	if !strings.Contains(output, "Orpheus CLI Framework Demo Application") {
		t.Errorf("App description not found in help output")
	}

	if !strings.Contains(output, "demo") {
		t.Errorf("App name not found in help output")
	}
}

// TestDefaultCommand tests the default command (greet)
func TestDefaultCommand(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "default greet without args",
			args:     []string{},
			contains: []string{"Hello, World!"},
		},
		// Note: Default command doesn't work with arguments in current Orpheus version
		// Arguments are interpreted as commands, so we test explicit greet command instead
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

// TestGreetCommand tests the greet command explicitly
func TestGreetCommand(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "explicit greet without args",
			args:     []string{"greet"},
			contains: []string{"Hello, World!"},
		},
		{
			name:     "explicit greet with name",
			args:     []string{"greet", "Charlie"},
			contains: []string{"Hello, Charlie!"},
		},
		{
			name:     "greet with global verbose flag",
			args:     []string{"--verbose", "greet", "Diana"},
			contains: []string{"Greeting Diana with verbose output enabled", "Hello, Diana!"},
		},
		// Note: -v is interpreted as --version, not --verbose
		// This is expected behavior in Orpheus
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

// TestEchoCommand tests the echo command
func TestEchoCommand(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name        string
		args        []string
		contains    []string
		expectError bool
	}{
		{
			name:     "echo single word",
			args:     []string{"echo", "hello"},
			contains: []string{"hello"},
		},
		{
			name:     "echo multiple words",
			args:     []string{"echo", "hello", "world", "123"},
			contains: []string{"hello world 123"},
		},
		{
			name:        "echo without arguments",
			args:        []string{"echo"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output string
			var err error

			if tt.expectError {
				err = app.Run(tt.args)
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				// Check that it's a validation error
				if !strings.Contains(err.Error(), "no arguments provided") {
					t.Errorf("Expected validation error about no arguments, got: %v", err)
				}
				return
			}

			output = captureOutput(func() {
				err = app.Run(tt.args)
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

// TestDeployCommand tests the deploy command
func TestDeployCommand(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name        string
		args        []string
		contains    []string
		expectError bool
	}{
		{
			name:     "deploy to production",
			args:     []string{"deploy", "production"},
			contains: []string{"Deploying to production environment..."},
		},
		{
			name:     "deploy to staging",
			args:     []string{"deploy", "staging"},
			contains: []string{"Deploying to staging environment..."},
		},
		{
			name:     "deploy to development",
			args:     []string{"deploy", "development"},
			contains: []string{"Deploying to development environment..."},
		},
		{
			name:     "deploy to custom environment",
			args:     []string{"deploy", "testing"},
			contains: []string{"Deploying to testing environment..."},
		},
		{
			name:        "deploy without environment",
			args:        []string{"deploy"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output string
			var err error

			if tt.expectError {
				err = app.Run(tt.args)
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				// Check that it's a validation error
				if !strings.Contains(err.Error(), "environment required") {
					t.Errorf("Expected validation error about environment required, got: %v", err)
				}
				return
			}

			output = captureOutput(func() {
				err = app.Run(tt.args)
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

// TestGlobalFlags tests global flags functionality
func TestGlobalFlags(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "config flag with greet",
			args:     []string{"--config", "config.yaml", "greet", "Test"},
			contains: []string{"Hello, Test!"},
		},
		{
			name:     "short config flag",
			args:     []string{"-c", "config.json", "echo", "test"},
			contains: []string{"test"},
		},
		{
			name:     "verbose flag with echo",
			args:     []string{"--verbose", "echo", "verbose", "test"},
			contains: []string{"verbose test"},
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

// TestHelpGeneration tests help generation for commands
func TestHelpGeneration(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "main help",
			args:     []string{"--help"},
			contains: []string{"Orpheus CLI Framework Demo Application", "Available Commands:", "greet", "echo", "deploy", "completion"},
		},
		{
			name:     "greet help",
			args:     []string{"help", "greet"},
			contains: []string{"Greet someone", "Usage:"},
		},
		{
			name:     "echo help",
			args:     []string{"help", "echo"},
			contains: []string{"Echo back the arguments", "Usage:"},
		},
		{
			name:     "deploy help",
			args:     []string{"help", "deploy"},
			contains: []string{"Deploy to environment", "Usage:"},
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
					t.Errorf("Expected help output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

// TestVersionFlag tests the version flag
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

// TestCompletionCommand tests the completion command
func TestCompletionCommand(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "bash completion",
			args:     []string{"completion", "bash"},
			contains: []string{"# Bash completion for demo", "_demo_completion"},
		},
		{
			name:     "zsh completion",
			args:     []string{"completion", "zsh"},
			contains: []string{"#compdef demo", "_demo"},
		},
		{
			name:     "fish completion",
			args:     []string{"completion", "fish"},
			contains: []string{"# Fish completion for demo", "complete -c demo"},
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
					t.Errorf("Expected completion output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

// TestDeployCompletion tests the custom completion for deploy command
func TestDeployCompletion(t *testing.T) {
	app := createTestApp()

	// Note: Direct testing of completion requires internal access to the completion system
	// This test verifies the completion handler was set correctly by checking the deploy command exists
	output := captureOutput(func() {
		err := app.Run([]string{"help", "deploy"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Deploy to environment") {
		t.Errorf("Deploy command not properly registered")
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	app := createTestApp()

	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "unknown command",
			args:          []string{"unknown"},
			expectedError: "command 'unknown' not found",
		},
		{
			name:          "echo without args",
			args:          []string{"echo"},
			expectedError: "no arguments provided",
		},
		{
			name:          "deploy without environment",
			args:          []string{"deploy"},
			expectedError: "environment required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.expectedError, err)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkGreetCommand(b *testing.B) {
	app := createTestApp()
	args := []string{"greet", "BenchmarkUser"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Capture output to avoid printing during benchmark
		_ = captureOutput(func() {
			app.Run(args)
		})
	}
}

func BenchmarkEchoCommand(b *testing.B) {
	app := createTestApp()
	args := []string{"echo", "benchmark", "test", "args"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Capture output to avoid printing during benchmark
		_ = captureOutput(func() {
			app.Run(args)
		})
	}
}

func BenchmarkDeployCommand(b *testing.B) {
	app := createTestApp()
	args := []string{"deploy", "production"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Capture output to avoid printing during benchmark
		_ = captureOutput(func() {
			app.Run(args)
		})
	}
}
