// orpheus_unit_test.go: unit tests for Orpheus application framework
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// Test missing coverage areas to reach 90%

func TestShowCommandHelp(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddFlag("file", "f", "input.txt", "Input file")
	app.AddCommand(cmd)

	// Test showing help for specific command
	err := app.Run([]string{"help", "test"})
	if err != nil {
		t.Errorf("expected no error showing command help, got %v", err)
	}
}

func TestGenerateHelp(t *testing.T) {
	app := orpheus.New("testapp")
	app.SetDescription("Test application")
	app.SetVersion("1.0.0")

	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddFlag("file", "f", "input.txt", "Input file")
	app.AddCommand(cmd)

	// Test general help
	err := app.Run([]string{"help"})
	if err != nil {
		t.Errorf("expected no error showing help, got %v", err)
	}

	// Test --help flag
	err = app.Run([]string{"--help"})
	if err != nil {
		t.Errorf("expected no error showing help with --help, got %v", err)
	}

	// Test -h flag
	err = app.Run([]string{"-h"})
	if err != nil {
		t.Errorf("expected no error showing help with -h, got %v", err)
	}
}

func TestSetCompletionHandler(t *testing.T) {
	cmd := orpheus.NewCommand("deploy", "Deploy application")

	// Test setting custom completion handler
	handler := func(req *orpheus.CompletionRequest) *orpheus.CompletionResult {
		return &orpheus.CompletionResult{
			Suggestions: []string{"production", "staging"},
		}
	}

	cmd.SetCompletionHandler(handler)

	// Verify the handler is set by testing completion
	app := orpheus.New("testapp")
	app.AddCommand(cmd)

	result := app.Complete([]string{"deploy", "prod"}, 2)
	if len(result.Suggestions) == 0 {
		t.Error("expected completion suggestions from custom handler")
	}
}

func TestGetGlobalFlagInt(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalIntFlag("port", "p", 8080, "Server port")

	cmd := orpheus.NewCommand("start", "Start server")
	app.AddCommand(cmd)

	// Test getting global int flag with default value
	var port int
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		port = ctx.GetGlobalFlagInt("port")
		return nil
	})

	err := app.Run([]string{"start"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if port != 8080 {
		t.Errorf("expected port = 8080 (default), got %d", port)
	}

	// Test with custom value
	err = app.Run([]string{"--port", "9000", "start"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if port != 9000 {
		t.Errorf("expected port = 9000, got %d", port)
	}

	// Test getting non-existent flag
	nonExistent := cmd
	nonExistent.SetHandler(func(ctx *orpheus.Context) error {
		port = ctx.GetGlobalFlagInt("nonexistent")
		return nil
	})

	err = app.Run([]string{"start"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGlobalFlagChanged(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalFlag("config", "c", "config.json", "Config file")

	cmd := orpheus.NewCommand("start", "Start server")
	app.AddCommand(cmd)

	// Test flag changed detection when flag is set
	var configChanged bool
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		configChanged = ctx.GlobalFlagChanged("config")
		return nil
	})

	err := app.Run([]string{"--config", "custom.json", "start"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !configChanged {
		t.Error("expected config flag to be detected as changed")
	}

	// Test non-existent flag
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		changed := ctx.GlobalFlagChanged("nonexistent")
		if changed {
			t.Error("expected non-existent flag to not be changed")
		}
		return nil
	})

	err = app.Run([]string{"start"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGetFlagFloat64(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddFloat64Flag("ratio", "r", 1.5, "Test ratio")
	app.AddCommand(cmd)

	// Test getting float64 flag
	var ratio float64
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		ratio = ctx.GetFlagFloat64("ratio")
		return nil
	})

	err := app.Run([]string{"test", "--ratio", "2.7"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if ratio != 2.7 {
		t.Errorf("expected ratio = 2.7, got %f", ratio)
	}
}

func TestGetFlagStringSlice(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddStringSliceFlag("tags", "t", []string{"default"}, "Tags list")
	app.AddCommand(cmd)

	// Test getting string slice flag
	var tags []string
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		tags = ctx.GetFlagStringSlice("tags")
		return nil
	})

	err := app.Run([]string{"test", "--tags", "tag1,tag2,tag3"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedTags := []string{"tag1", "tag2", "tag3"}
	if len(tags) != len(expectedTags) {
		t.Errorf("expected %d tags, got %d", len(expectedTags), len(tags))
	}

	for i, expected := range expectedTags {
		if i >= len(tags) || tags[i] != expected {
			t.Errorf("expected tag[%d] = '%s', got '%s'", i, expected, tags[i])
		}
	}
}

func TestAddCompletionCommandFullCoverage(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddCompletionCommand()

	// Test generating bash completion
	err := app.Run([]string{"completion", "bash"})
	if err != nil {
		t.Errorf("expected no error for bash completion, got %v", err)
	}

	// Test generating zsh completion
	err = app.Run([]string{"completion", "zsh"})
	if err != nil {
		t.Errorf("expected no error for zsh completion, got %v", err)
	}

	// Test generating fish completion
	err = app.Run([]string{"completion", "fish"})
	if err != nil {
		t.Errorf("expected no error for fish completion, got %v", err)
	}
}

func TestCompleteWithDifferentScenarios(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalFlag("verbose", "v", "false", "Verbose output")

	cmd := orpheus.NewCommand("deploy", "Deploy application")
	cmd.AddFlag("env", "e", "prod", "Environment")
	app.AddCommand(cmd)

	// Test completing global flags
	result := app.Complete([]string{"deploy", "--v"}, 2)
	found := false
	for _, suggestion := range result.Suggestions {
		if strings.Contains(suggestion, "verbose") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected --verbose in global flag completion")
	}

	// Test completing at position beyond args length
	result = app.Complete([]string{"deploy"}, 2)
	if result == nil {
		t.Error("expected completion result")
	}
}

func TestGetFlagMethods(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("test", "Test command")

	// Test GetFlag method
	cmd.AddFlag("name", "n", "default", "Name flag")
	app.AddCommand(cmd)

	var flagValue interface{}
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		flagValue = ctx.GetFlag("name")
		return nil
	})

	err := app.Run([]string{"test", "--name", "testvalue"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if flagValue != "testvalue" {
		t.Errorf("expected flagValue = 'testvalue', got %v", flagValue)
	}
}

func TestGetGlobalFlag(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalFlag("debug", "d", "false", "Debug mode")

	cmd := orpheus.NewCommand("test", "Test command")
	app.AddCommand(cmd)

	var debugValue interface{}
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		debugValue = ctx.GetGlobalFlag("debug")
		return nil
	})

	err := app.Run([]string{"--debug", "true", "test"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if debugValue != "true" {
		t.Errorf("expected debugValue = 'true', got %v", debugValue)
	}
}
