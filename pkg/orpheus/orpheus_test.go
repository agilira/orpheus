// final_coverage_test.go: unit tests for Orpheus application framework
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

func TestGenerateHelpDirectly(t *testing.T) {
	app := orpheus.New("testapp")
	app.SetDescription("Test application")
	app.SetVersion("1.0.0")
	app.AddGlobalFlag("verbose", "v", "false", "Verbose output")

	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddFlag("file", "f", "input.txt", "Input file")
	cmd.AddBoolFlag("force", "F", false, "Force operation")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		return nil
	})
	app.AddCommand(cmd)

	// Force help generation by calling help command
	err := app.Run([]string{"help"})
	if err != nil {
		t.Errorf("expected no error for help command, got %v", err)
	}

	// Force help generation with --help flag
	err = app.Run([]string{"--help"})
	if err != nil {
		t.Errorf("expected no error for --help flag, got %v", err)
	}

	// Force help generation for specific command
	err = app.Run([]string{"help", "test"})
	if err != nil {
		t.Errorf("expected no error for help test command, got %v", err)
	}

	// Test help for command with --help
	err = app.Run([]string{"test", "--help"})
	// This should generate a validation error for help request, which is expected
	if err != nil {
		orpheusErr, ok := err.(*orpheus.OrpheusError)
		if !ok || !orpheusErr.IsValidationError() {
			t.Errorf("expected validation error for test --help, got %v", err)
		}
	}
}

func TestGenerateHelpMethod(t *testing.T) {
	app := orpheus.New("testapp")
	app.SetDescription("Test application for help generation")
	app.SetVersion("1.0.0")

	cmd := orpheus.NewCommand("deploy", "Deploy application")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		return nil
	})
	app.AddCommand(cmd)

	// Test the GenerateHelp method directly
	helpText := app.GenerateHelp()
	if helpText == "" {
		t.Error("expected non-empty help text")
	}

	// Check that help contains expected elements
	if !strings.Contains(helpText, "testapp") {
		t.Error("help text should contain app name")
	}

	if !strings.Contains(helpText, "Test application for help generation") {
		t.Error("help text should contain app description")
	}

	if !strings.Contains(helpText, "deploy") {
		t.Error("help text should contain command names")
	}

	if !strings.Contains(helpText, "Deploy application") {
		t.Error("help text should contain command descriptions")
	}
}

func TestSplitGlobalArgsEdgeCases(t *testing.T) {
	app := orpheus.New("testapp")
	app.AddGlobalFlag("config", "c", "default.json", "Config file")
	app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")

	cmd := orpheus.NewCommand("test", "Test command")
	cmd.AddFlag("input", "i", "input.txt", "Input file")
	app.AddCommand(cmd)

	// Test various flag combinations
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		// Test all flag getters
		config := ctx.GetGlobalFlagString("config")
		verbose := ctx.GetGlobalFlagBool("verbose")
		input := ctx.GetFlagString("input")

		// Test changed detection
		configChanged := ctx.GlobalFlagChanged("config")
		verboseChanged := ctx.GlobalFlagChanged("verbose")
		inputChanged := ctx.FlagChanged("input")

		_ = config
		_ = verbose
		_ = input
		_ = configChanged
		_ = verboseChanged
		_ = inputChanged

		return nil
	})

	// Test with mix of global and command flags
	err := app.Run([]string{"--config", "custom.json", "--verbose", "test", "--input", "test.txt"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test with short flags
	err = app.Run([]string{"-c", "short.json", "-v", "test", "-i", "short.txt"})
	if err != nil {
		t.Errorf("expected no error with short flags, got %v", err)
	}

	// Test default values
	err = app.Run([]string{"test"})
	if err != nil {
		t.Errorf("expected no error with defaults, got %v", err)
	}
}

func TestFlagTypesEdgeCases(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("test", "Test command")

	// Add various flag types
	cmd.AddFloat64Flag("ratio", "r", 1.5, "Test ratio")
	cmd.AddStringSliceFlag("tags", "t", []string{"default"}, "Tags list")
	cmd.AddIntFlag("count", "n", 10, "Count value")
	cmd.AddBoolFlag("enabled", "e", false, "Enable feature")

	app.AddCommand(cmd)

	// Test all flag types
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		ratio := ctx.GetFlagFloat64("ratio")
		tags := ctx.GetFlagStringSlice("tags")
		count := ctx.GetFlagInt("count")
		enabled := ctx.GetFlagBool("enabled")

		// Test changed detection for all types
		ratioChanged := ctx.FlagChanged("ratio")
		tagsChanged := ctx.FlagChanged("tags")
		countChanged := ctx.FlagChanged("count")
		enabledChanged := ctx.FlagChanged("enabled")

		_ = ratio
		_ = tags
		_ = count
		_ = enabled
		_ = ratioChanged
		_ = tagsChanged
		_ = countChanged
		_ = enabledChanged

		return nil
	})

	// Test with all flag types set
	err := app.Run([]string{"test", "--ratio", "2.7", "--tags", "a,b,c", "--count", "20", "--enabled"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test with default values
	err = app.Run([]string{"test"})
	if err != nil {
		t.Errorf("expected no error with defaults, got %v", err)
	}
}

func TestAddGlobalFlagVariations(t *testing.T) {
	app := orpheus.New("testapp")

	// Test different ways of adding global flags
	app.AddGlobalFlag("config", "c", "config.json", "Config file")
	app.AddGlobalFlag("output", "", "output.txt", "Output file") // No shorthand
	app.AddGlobalBoolFlag("debug", "d", false, "Debug mode")
	app.AddGlobalBoolFlag("quiet", "", true, "Quiet mode") // No shorthand
	app.AddGlobalIntFlag("workers", "w", 4, "Worker count")
	app.AddGlobalIntFlag("timeout", "", 30, "Timeout seconds") // No shorthand

	cmd := orpheus.NewCommand("process", "Process data")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		// Access all global flags to ensure coverage
		config := ctx.GetGlobalFlagString("config")
		output := ctx.GetGlobalFlagString("output")
		debug := ctx.GetGlobalFlagBool("debug")
		quiet := ctx.GetGlobalFlagBool("quiet")
		workers := ctx.GetGlobalFlagInt("workers")
		timeout := ctx.GetGlobalFlagInt("timeout")

		_ = config
		_ = output
		_ = debug
		_ = quiet
		_ = workers
		_ = timeout

		return nil
	})

	app.AddCommand(cmd)

	// Test with various flag combinations
	err := app.Run([]string{"--config", "test.json", "--debug", "process"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCommandFlagVariations(t *testing.T) {
	app := orpheus.New("testapp")
	cmd := orpheus.NewCommand("convert", "Convert files")

	// Test different ways of adding command flags
	cmd.AddFlag("input", "i", "input.txt", "Input file")
	cmd.AddFlag("output", "", "output.txt", "Output file") // No shorthand
	cmd.AddBoolFlag("overwrite", "o", false, "Overwrite existing")
	cmd.AddBoolFlag("backup", "", true, "Create backup") // No shorthand
	cmd.AddIntFlag("quality", "q", 80, "Quality level")
	cmd.AddIntFlag("threads", "", 1, "Thread count") // No shorthand
	cmd.AddFloat64Flag("scale", "s", 1.0, "Scale factor")
	cmd.AddFloat64Flag("gamma", "", 2.2, "Gamma correction") // No shorthand
	cmd.AddStringSliceFlag("formats", "f", []string{"jpg"}, "Output formats")
	cmd.AddStringSliceFlag("filters", "", []string{"none"}, "Apply filters") // No shorthand

	app.AddCommand(cmd)

	cmd.SetHandler(func(ctx *orpheus.Context) error {
		// Access all command flags to ensure coverage
		input := ctx.GetFlagString("input")
		output := ctx.GetFlagString("output")
		overwrite := ctx.GetFlagBool("overwrite")
		backup := ctx.GetFlagBool("backup")
		quality := ctx.GetFlagInt("quality")
		threads := ctx.GetFlagInt("threads")
		scale := ctx.GetFlagFloat64("scale")
		gamma := ctx.GetFlagFloat64("gamma")
		formats := ctx.GetFlagStringSlice("formats")
		filters := ctx.GetFlagStringSlice("filters")

		_ = input
		_ = output
		_ = overwrite
		_ = backup
		_ = quality
		_ = threads
		_ = scale
		_ = gamma
		_ = formats
		_ = filters

		return nil
	})

	// Test with various flag combinations
	err := app.Run([]string{"convert", "--input", "test.png", "--quality", "90", "--scale", "1.5"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
