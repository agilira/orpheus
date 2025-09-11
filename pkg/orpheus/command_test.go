// command_test.go: tests for commands in orpheus
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

func TestCommandCreation(t *testing.T) {
	cmd := orpheus.NewCommand("test", "Test command")

	if cmd.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", cmd.Name())
	}

	if cmd.Description() != "Test command" {
		t.Errorf("expected description 'Test command', got '%s'", cmd.Description())
	}

	if cmd.Usage() != "test [flags]" {
		t.Errorf("expected usage 'test [flags]', got '%s'", cmd.Usage())
	}
}

func TestCommandHandler(t *testing.T) {
	var executed bool
	var receivedCtx *orpheus.Context

	cmd := orpheus.NewCommand("test", "Test command").
		SetHandler(func(ctx *orpheus.Context) error {
			executed = true
			receivedCtx = ctx
			return nil
		})

	ctx := &orpheus.Context{
		Args: []string{"arg1", "arg2"},
	}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !executed {
		t.Error("handler was not executed")
	}

	if receivedCtx != ctx {
		t.Error("handler did not receive correct context")
	}
}

func TestCommandWithCustomUsage(t *testing.T) {
	cmd := orpheus.NewCommand("copy", "Copy files").
		SetUsage("copy <source> <destination>")

	if cmd.Usage() != "copy <source> <destination>" {
		t.Errorf("expected custom usage, got '%s'", cmd.Usage())
	}
}

func TestCommandExecution(t *testing.T) {
	var argCount int
	var firstArg string

	cmd := orpheus.NewCommand("test", "Test command").
		SetHandler(func(ctx *orpheus.Context) error {
			argCount = ctx.ArgCount()
			if argCount > 0 {
				firstArg = ctx.GetArg(0)
			}
			return nil
		})

	ctx := &orpheus.Context{
		Args: []string{"hello", "world"},
	}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if argCount != 2 {
		t.Errorf("expected 2 args, got %d", argCount)
	}

	if firstArg != "hello" {
		t.Errorf("expected first arg 'hello', got '%s'", firstArg)
	}
}

func TestCommandError(t *testing.T) {
	cmd := orpheus.NewCommand("fail", "Failing command").
		SetHandler(func(ctx *orpheus.Context) error {
			return orpheus.ValidationError("fail", "test error")
		})

	ctx := &orpheus.Context{}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("expected error from failing command")
	}

	orpheusErr, ok := err.(*orpheus.OrpheusError)
	if !ok {
		t.Errorf("expected OrpheusError, got %T", err)
	}

	if !orpheusErr.IsValidationError() {
		t.Error("expected ValidationError")
	}
}

func TestCommandAutoHelp(t *testing.T) {
	var output string
	cmd := orpheus.NewCommand("test", "Test command").
		SetHandler(func(ctx *orpheus.Context) error {
			t.Error("handler should not be called when help is requested")
			return nil
		})

	// Create a test app to provide context
	app := orpheus.New("testapp")
	app.AddCommand(cmd)

	// Test --help
	ctx := &orpheus.Context{
		App:  app,
		Args: []string{"--help"},
	}

	err := cmd.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error for --help, got: %v", err)
	}

	// Test -h
	ctx = &orpheus.Context{
		App:  app,
		Args: []string{"-h"},
	}

	err = cmd.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error for -h, got: %v", err)
	}

	// Test that normal execution still works
	normalCmd := orpheus.NewCommand("normal", "Normal command").
		SetHandler(func(ctx *orpheus.Context) error {
			output = "handler executed"
			return nil
		})

	ctx = &orpheus.Context{
		App:  app,
		Args: []string{},
	}

	err = normalCmd.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error for normal execution, got: %v", err)
	}
	if output != "handler executed" {
		t.Error("handler should have been executed for normal command")
	}
}

func TestCommandNoHandler(t *testing.T) {
	cmd := orpheus.NewCommand("test", "Test command")
	// Don't set a handler

	ctx := &orpheus.Context{}

	err := cmd.Execute(ctx)
	if err == nil {
		t.Error("expected error when no handler is set")
	}
}
