// integrations_test.go: integration tests for Orpheus
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

func TestFlashFlagsIntegration(t *testing.T) {
	app := orpheus.New("testapp")

	// Add a command with flags
	cmd := orpheus.NewCommand("server", "Start the server").
		SetHandler(func(ctx *orpheus.Context) error {
			host := ctx.GetFlagString("host")
			port := ctx.GetFlagInt("port")
			debug := ctx.GetFlagBool("debug")

			if host == "" {
				return orpheus.ValidationError("server", "host is required")
			}

			if port < 1 || port > 65535 {
				return orpheus.ValidationError("server", "port must be between 1 and 65535")
			}

			// Test that we got the expected values
			if host != "localhost" {
				return orpheus.ValidationError("server", "unexpected host value")
			}
			if port != 8080 {
				return orpheus.ValidationError("server", "unexpected port value")
			}
			if !debug {
				return orpheus.ValidationError("server", "expected debug to be true")
			}

			return nil
		})

	// Add flags to the command
	cmd.AddFlag("host", "h", "localhost", "Server host").
		AddIntFlag("port", "p", 8080, "Server port").
		AddBoolFlag("debug", "d", false, "Enable debug mode")

	app.AddCommand(cmd)

	// Test flag parsing
	err := app.Run([]string{"server", "--host", "localhost", "--port", "8080", "--debug"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGlobalFlagsIntegration(t *testing.T) {
	app := orpheus.New("testapp")

	// Add global flags
	app.AddGlobalFlag("config", "c", "config.json", "Configuration file").
		AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output")

	var receivedConfig string
	var receivedVerbose bool

	app.Command("test", "Test command", func(ctx *orpheus.Context) error {
		receivedConfig = ctx.GetGlobalFlagString("config")
		receivedVerbose = ctx.GetGlobalFlagBool("verbose")
		return nil
	})

	// Test global flag parsing
	err := app.Run([]string{"--config", "test.json", "--verbose", "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedConfig != "test.json" {
		t.Errorf("expected config 'test.json', got '%s'", receivedConfig)
	}

	if !receivedVerbose {
		t.Errorf("expected verbose to be true")
	}
}

func TestFlagChangedDetection(t *testing.T) {
	app := orpheus.New("testapp")

	var hostChanged, portChanged bool

	cmd := orpheus.NewCommand("server", "Start server").
		SetHandler(func(ctx *orpheus.Context) error {
			hostChanged = ctx.FlagChanged("host")
			portChanged = ctx.FlagChanged("port")
			return nil
		})

	cmd.AddFlag("host", "h", "localhost", "Server host").
		AddIntFlag("port", "p", 8080, "Server port")

	app.AddCommand(cmd)

	// Only set host flag, not port
	err := app.Run([]string{"server", "--host", "example.com"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !hostChanged {
		t.Error("expected host flag to be marked as changed")
	}

	if portChanged {
		t.Error("expected port flag to NOT be marked as changed")
	}
}

func TestFlagTypesIntegration(t *testing.T) {
	app := orpheus.New("testapp")

	var receivedString string
	var receivedInt int
	var receivedBool bool

	cmd := orpheus.NewCommand("test", "Test all flag types").
		SetHandler(func(ctx *orpheus.Context) error {
			receivedString = ctx.GetFlagString("str")
			receivedInt = ctx.GetFlagInt("num")
			receivedBool = ctx.GetFlagBool("flag")
			return nil
		})

	// Add different flag types
	cmd.AddFlag("str", "", "default", "String flag").
		AddIntFlag("num", "", 0, "Integer flag").
		AddBoolFlag("flag", "", false, "Boolean flag")

	app.AddCommand(cmd)

	// Test basic flag types
	err := app.Run([]string{"test", "--str", "hello", "--num", "42", "--flag"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedString != "hello" {
		t.Errorf("expected string 'hello', got '%s'", receivedString)
	}

	if receivedInt != 42 {
		t.Errorf("expected int 42, got %d", receivedInt)
	}

	if !receivedBool {
		t.Error("expected bool to be true")
	}
}

func TestHelpGenerationWithFlags(t *testing.T) {
	app := orpheus.New("testapp").
		SetDescription("Test application").
		SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalFlag("config", "c", "config.json", "Configuration file").
		AddGlobalBoolFlag("verbose", "v", false, "Enable verbose output")

	// Add command with flags
	cmd := orpheus.NewCommand("server", "Start the server").
		SetLongDescription("Start the HTTP server with the specified configuration.").
		AddExample("testapp server --host localhost --port 8080").
		AddExample("testapp server --config production.json").
		SetHandler(func(ctx *orpheus.Context) error { return nil })

	cmd.AddFlag("host", "h", "localhost", "Server host").
		AddIntFlag("port", "p", 8080, "Server port").
		AddBoolFlag("debug", "d", false, "Enable debug mode")

	app.AddCommand(cmd)

	// Test help generation
	generator := app.GetHelpGenerator()
	help := generator.GenerateCommandHelp(cmd)

	// Check that help contains expected elements
	if !strings.Contains(help, "Start the HTTP server") {
		t.Error("help should contain long description")
	}

	if !strings.Contains(help, "testapp server --host") {
		t.Error("help should contain examples")
	}

	// Note: Flag help will be tested once flash-flags introspection is fully working
}

func TestCompletionWithFlags(t *testing.T) {
	app := orpheus.New("testapp").SetVersion("1.0.0")

	// Add global flags
	app.AddGlobalFlag("config", "c", "config.json", "Configuration file")

	// Add command with flags
	cmd := orpheus.NewCommand("server", "Start server")
	cmd.AddFlag("host", "h", "localhost", "Server host").
		AddIntFlag("port", "p", 8080, "Server port")

	app.AddCommand(cmd)

	// Test flag completion
	result := app.Complete([]string{"server", "--h"}, 2)

	// Should suggest flags starting with --h
	found := false
	for _, suggestion := range result.Suggestions {
		if suggestion == "--help" || suggestion == "--host" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected --help or --host in suggestions, got %v", result.Suggestions)
	}
}
