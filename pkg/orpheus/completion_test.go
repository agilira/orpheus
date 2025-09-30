// completion_test.go: auto-completion tests
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

func TestCommandCompletion(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
		return nil
	})
	app.Command("stop", "Stop the service", func(ctx *orpheus.Context) error {
		return nil
	})
	app.Command("status", "Show service status", func(ctx *orpheus.Context) error {
		return nil
	})

	// Test completing commands with no input
	result := app.Complete([]string{}, 0)
	if len(result.Suggestions) != 4 { // start, stop, status, help
		t.Errorf("expected 4 suggestions, got %d", len(result.Suggestions))
	}

	expectedCommands := []string{"help", "start", "status", "stop"}
	for i, expected := range expectedCommands {
		if result.Suggestions[i] != expected {
			t.Errorf("expected suggestion[%d] = '%s', got '%s'", i, expected, result.Suggestions[i])
		}
	}
}

func TestPartialCommandCompletion(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
		return nil
	})
	app.Command("stop", "Stop the service", func(ctx *orpheus.Context) error {
		return nil
	})
	app.Command("status", "Show service status", func(ctx *orpheus.Context) error {
		return nil
	})

	// Test completing commands starting with "st"
	result := app.Complete([]string{"st"}, 1)
	if len(result.Suggestions) != 3 { // start, stop, status
		t.Errorf("expected 3 suggestions, got %d", len(result.Suggestions))
	}

	expectedCommands := []string{"start", "status", "stop"}
	for i, expected := range expectedCommands {
		if result.Suggestions[i] != expected {
			t.Errorf("expected suggestion[%d] = '%s', got '%s'", i, expected, result.Suggestions[i])
		}
	}
}

func TestFlagCompletion(t *testing.T) {
	app := orpheus.New("testapp").SetVersion("1.0.0")
	cmd := orpheus.NewCommand("test", "Test command")
	app.AddCommand(cmd)

	// Test basic completion functionality
	result := app.Complete([]string{"test", "--h"}, 2)

	if result == nil {
		t.Error("expected completion result, got nil")
	}
}

func TestBashCompletionGeneration(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
		return nil
	})
	app.Command("stop", "Stop the service", func(ctx *orpheus.Context) error {
		return nil
	})

	script := app.GenerateCompletion("bash")

	// Check that the script contains expected elements
	if !strings.Contains(script, "testapp") {
		t.Error("bash script should contain app name")
	}

	if !strings.Contains(script, "_testapp_completion") {
		t.Error("bash script should contain completion function")
	}

	if !strings.Contains(script, "start stop") {
		t.Error("bash script should contain command names")
	}

	if !strings.Contains(script, "complete -F") {
		t.Error("bash script should contain complete command")
	}
}

func TestZshCompletionGeneration(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
		return nil
	})

	script := app.GenerateCompletion("zsh")

	// Check that the script contains expected elements
	if !strings.Contains(script, "#compdef testapp") {
		t.Error("zsh script should contain compdef directive")
	}

	if !strings.Contains(script, "_testapp()") {
		t.Error("zsh script should contain completion function")
	}

	if !strings.Contains(script, "start:'Start the service'") {
		t.Error("zsh script should contain command descriptions")
	}
}

func TestFishCompletionGeneration(t *testing.T) {
	app := orpheus.New("testapp")
	app.Command("start", "Start the service", func(ctx *orpheus.Context) error {
		return nil
	})

	script := app.GenerateCompletion("fish")

	// Check that the script contains expected elements
	if !strings.Contains(script, "complete -c testapp") {
		t.Error("fish script should contain complete commands")
	}

	if !strings.Contains(script, "-a start") {
		t.Error("fish script should contain command definitions")
	}

	if !strings.Contains(script, "-d 'Start the service'") {
		t.Error("fish script should contain command descriptions")
	}
}

func TestCompletionCommand(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddCompletionCommand()

	// Check that completion command was added
	commands := app.GetCommands()
	if _, exists := commands["completion"]; !exists {
		t.Error("completion command should be added")
	}
}

func TestGenerateCompletionDefault(t *testing.T) {
	app := orpheus.New("testapp")

	// Test default case (should default to bash)
	unknownShell := app.GenerateCompletion("unknown")
	bashCompletion := app.GenerateCompletion("bash")

	if unknownShell != bashCompletion {
		t.Error("unknown shell completion should default to bash")
	}

	// Ensure it's actually bash completion
	if !strings.Contains(unknownShell, "_completion() {") {
		t.Error("default completion should be bash format")
	}
}
