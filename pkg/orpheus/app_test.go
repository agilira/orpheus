// app_test.go: tests for the main CLI application
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"os"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestAppCreation(t *testing.T) {
	app := orpheus.New("testapp")

	if app.Name() != "testapp" {
		t.Errorf("expected app name 'testapp', got '%s'", app.Name())
	}

	if app.Version() != "" {
		t.Errorf("expected empty version, got '%s'", app.Version())
	}

	if app.Description() != "" {
		t.Errorf("expected empty description, got '%s'", app.Description())
	}
}

func TestAppConfiguration(t *testing.T) {
	app := orpheus.New("testapp").
		SetDescription("Test application").
		SetVersion("1.0.0")

	if app.Description() != "Test application" {
		t.Errorf("expected description 'Test application', got '%s'", app.Description())
	}

	if app.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", app.Version())
	}
}

func TestCommandRegistration(t *testing.T) {
	var executed bool
	var receivedCtx *orpheus.Context

	app := orpheus.New("testapp")
	app.Command("test", "Test command", func(ctx *orpheus.Context) error {
		executed = true
		receivedCtx = ctx
		return nil
	})

	commands := app.GetCommands()
	if len(commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(commands))
	}

	if _, exists := commands["test"]; !exists {
		t.Error("expected 'test' command to exist")
	}

	// Test command execution
	err := app.Run([]string{"test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed {
		t.Error("command was not executed")
	}

	if receivedCtx == nil {
		t.Error("context was not passed to handler")
	}

	if receivedCtx.App != app {
		t.Error("context does not contain correct app reference")
	}
}

func TestCommandWithArgs(t *testing.T) {
	var receivedArgs []string

	app := orpheus.New("testapp")
	app.Command("echo", "Echo arguments", func(ctx *orpheus.Context) error {
		receivedArgs = make([]string, ctx.ArgCount())
		for i := 0; i < ctx.ArgCount(); i++ {
			receivedArgs[i] = ctx.GetArg(i)
		}
		return nil
	})

	err := app.Run([]string{"echo", "hello", "world"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedArgs := []string{"hello", "world"}
	if len(receivedArgs) != len(expectedArgs) {
		t.Errorf("expected %d args, got %d", len(expectedArgs), len(receivedArgs))
	}

	for i, expected := range expectedArgs {
		if receivedArgs[i] != expected {
			t.Errorf("expected arg[%d] = '%s', got '%s'", i, expected, receivedArgs[i])
		}
	}
}

func TestDefaultCommand(t *testing.T) {
	var executed bool

	app := orpheus.New("testapp")
	app.Command("default", "Default command", func(ctx *orpheus.Context) error {
		executed = true
		return nil
	})
	app.SetDefaultCommand("default")

	err := app.Run([]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed {
		t.Error("default command was not executed")
	}
}

func TestCommandNotFound(t *testing.T) {
	app := orpheus.New("testapp")

	err := app.Run([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent command")
	}

	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Errorf("expected OrpheusError, got %T", err)
	}

	if !orpheusErr.IsNotFoundError() {
		t.Error("expected NotFoundError")
	}
}

func TestVersionFlag(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := orpheus.New("testapp").SetVersion("1.2.3")
	err := app.Run([]string{"--version"})

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expected := "testapp version 1.2.3\n"
	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
}

func TestHelpGeneration(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := orpheus.New("testapp").
		SetDescription("Test application for help").
		SetVersion("1.0.0")

	app.Command("cmd1", "First command", func(ctx *orpheus.Context) error {
		return nil
	})

	app.Command("cmd2", "Second command", func(ctx *orpheus.Context) error {
		return nil
	})

	err := app.Run([]string{"--help"})

	if closeErr := w.Close(); closeErr != nil {
		t.Errorf("failed to close pipe: %v", closeErr)
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that help contains expected elements
	if !strings.Contains(output, "Test application for help") {
		t.Error("help should contain app description")
	}

	if !strings.Contains(output, "Usage: testapp [command] [flags]") {
		t.Error("help should contain usage line")
	}

	if !strings.Contains(output, "cmd1") {
		t.Error("help should contain cmd1")
	}

	if !strings.Contains(output, "cmd2") {
		t.Error("help should contain cmd2")
	}

	if !strings.Contains(output, "First command") {
		t.Error("help should contain cmd1 description")
	}

	if !strings.Contains(output, "--help") {
		t.Error("help should contain --help flag")
	}

	if !strings.Contains(output, "--version") {
		t.Error("help should contain --version flag")
	}
}

func TestErrorHandling(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("fail", "Failing command", func(ctx *orpheus.Context) error {
		return orpheus.ExecutionError("fail", "something went wrong")
	})

	err := app.Run([]string{"fail"})
	if err == nil {
		t.Error("expected error from failing command")
	}

	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Errorf("expected OrpheusError, got %T", err)
	}

	if !orpheusErr.IsExecutionError() {
		t.Error("expected ExecutionError")
	}

	if orpheusErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", orpheusErr.ExitCode())
	}
}

func TestGlobalFlags(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output")
	app.AddGlobalFlag("config", "c", "default.conf", "Config file")

	// This test verifies that global flags can be added without errors
	// Full flag parsing will be tested when flash-flags integration is complete

	commands := app.GetCommands()
	if len(commands) != 0 {
		t.Errorf("expected 0 commands after adding global flags, got %d", len(commands))
	}
}
