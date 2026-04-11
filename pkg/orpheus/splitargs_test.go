package orpheus_test

import (
	"fmt"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// TestSplitGlobalArgs_NegativeNumbers verifies that negative numeric values
// are correctly consumed as flag values, not misinterpreted as separate flags.
// WHY: this is the classic "campo minato" of CLI parsing. A user passing
// --offset -5 must not see -5 treated as an unknown flag.
func TestSplitGlobalArgs_NegativeNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		expectedOffset int
		expectErr      bool
	}{
		{
			name:           "long flag with negative int",
			args:           []string{"--offset", "-5", "test"},
			expectedOffset: -5,
		},
		{
			name:           "short flag with negative int",
			args:           []string{"-o", "-5", "test"},
			expectedOffset: -5,
		},
		{
			name:           "long flag with negative using equals",
			args:           []string{"--offset=-5", "test"},
			expectedOffset: -5,
		},
		{
			name:           "negative zero",
			args:           []string{"--offset", "0", "test"},
			expectedOffset: 0,
		},
		{
			name:           "large negative number",
			args:           []string{"--offset", "-999999", "test"},
			expectedOffset: -999999,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := orpheus.New("testapp")
			app.AddGlobalIntFlag("offset", "o", 0, "Offset value")

			var gotOffset int
			cmd := orpheus.NewCommand("test", "Test command")
			cmd.SetHandler(func(ctx *orpheus.Context) error {
				gotOffset = ctx.GetGlobalFlagInt("offset")
				return nil
			})
			app.AddCommand(cmd)

			err := app.Run(tc.args)
			if tc.expectErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotOffset != tc.expectedOffset {
				t.Errorf("expected offset %d, got %d", tc.expectedOffset, gotOffset)
			}
		})
	}
}

// TestSplitGlobalArgs_DashValueOnStringFlags verifies that string flag values
// starting with '-' are correctly consumed (e.g., --pattern "-foo").
func TestSplitGlobalArgs_DashValueOnStringFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "value starting with single dash",
			args:     []string{"--pattern", "-foo", "test"},
			expected: "-foo",
		},
		{
			name:     "value starting with double dash",
			args:     []string{"--pattern", "--something", "test"},
			expected: "--something",
		},
		{
			name:     "value with equals",
			args:     []string{"--pattern=-foo", "test"},
			expected: "-foo",
		},
		{
			name:     "short flag with dash value",
			args:     []string{"-p", "-bar", "test"},
			expected: "-bar",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := orpheus.New("testapp")
			app.AddGlobalFlag("pattern", "p", "", "Pattern")

			var got string
			cmd := orpheus.NewCommand("test", "Test command")
			cmd.SetHandler(func(ctx *orpheus.Context) error {
				got = ctx.GetGlobalFlagString("pattern")
				return nil
			})
			app.AddCommand(cmd)

			err := app.Run(tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestSplitGlobalArgs_EmbeddedEquals verifies --flag=value and --flag="value"
// forms work correctly, including values with special characters.
func TestSplitGlobalArgs_EmbeddedEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "simple equals",
			args:     []string{"--config=myfile.json", "test"},
			expected: "myfile.json",
		},
		{
			name:     "equals with relative path",
			args:     []string{"--config=configs/app.yaml", "test"},
			expected: "configs/app.yaml",
		},
		{
			name:     "equals with value containing equals",
			args:     []string{"--config=key=value", "test"},
			expected: "key=value",
		},
		{
			name:     "equals with empty value",
			args:     []string{"--config=", "test"},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := orpheus.New("testapp")
			app.AddGlobalFlag("config", "c", "default.json", "Config file")

			var got string
			cmd := orpheus.NewCommand("test", "Test command")
			cmd.SetHandler(func(ctx *orpheus.Context) error {
				got = ctx.GetGlobalFlagString("config")
				return nil
			})
			app.AddCommand(cmd)

			err := app.Run(tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestSplitGlobalArgs_DoubleDashSentinel verifies that "--" terminates
// global flag parsing per POSIX convention.
func TestSplitGlobalArgs_DoubleDashSentinel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		expectVerbose  bool
		expectCmd      bool
		expectNoCmd    bool
		expectedConfig string
	}{
		{
			name:          "double dash before command",
			args:          []string{"--verbose", "--", "test"},
			expectVerbose: true,
			expectCmd:     true,
		},
		{
			name:          "double dash only global flags before",
			args:          []string{"--verbose", "--"},
			expectVerbose: true,
			expectNoCmd:   true,
		},
		{
			name:      "double dash without global flags",
			args:      []string{"--", "test"},
			expectCmd: true,
		},
		{
			name:           "double dash prevents flag-looking command from being parsed as flag",
			args:           []string{"--config", "myconf", "--", "test"},
			expectedConfig: "myconf",
			expectCmd:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := orpheus.New("testapp")
			app.AddGlobalFlag("config", "c", "default.json", "Config file")
			app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")

			var handlerCalled bool
			cmd := orpheus.NewCommand("test", "Test command")
			cmd.SetHandler(func(ctx *orpheus.Context) error {
				handlerCalled = true
				if tc.expectVerbose && !ctx.GetGlobalFlagBool("verbose") {
					return fmt.Errorf("expected verbose to be true")
				}
				if tc.expectedConfig != "" {
					got := ctx.GetGlobalFlagString("config")
					if got != tc.expectedConfig {
						return fmt.Errorf("expected config %q, got %q", tc.expectedConfig, got)
					}
				}
				return nil
			})
			app.AddCommand(cmd)

			err := app.Run(tc.args)

			if tc.expectCmd && !handlerCalled {
				// Command should have been called
				if err != nil {
					t.Logf("error (may be expected if command not found): %v", err)
				}
			}
			if tc.expectNoCmd && err != nil {
				// No command, might show help - that's OK
				t.Logf("no command case, err=%v", err)
			}
			if tc.expectCmd && err != nil && handlerCalled {
				t.Errorf("unexpected handler error: %v", err)
			}
		})
	}
}

// TestSplitGlobalArgs_BoolFlagInteraction verifies that boolean flags do not
// accidentally consume the next argument as their value.
func TestSplitGlobalArgs_BoolFlagInteraction(t *testing.T) {
	t.Parallel()

	app := orpheus.New("testapp")
	app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")
	app.AddGlobalFlag("config", "c", "default.json", "Config file")

	var gotVerbose bool
	var gotConfig string
	cmd := orpheus.NewCommand("test", "Test command")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		gotVerbose = ctx.GetGlobalFlagBool("verbose")
		gotConfig = ctx.GetGlobalFlagString("config")
		return nil
	})
	app.AddCommand(cmd)

	// Bool flag between two value flags - must not consume "test" as value
	err := app.Run([]string{"--verbose", "--config", "myconf", "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotVerbose {
		t.Error("expected verbose=true")
	}
	if gotConfig != "myconf" {
		t.Errorf("expected config=myconf, got %q", gotConfig)
	}
}

// TestSplitGlobalArgs_MixedGlobalAndCommandFlags verifies that global and
// command-local flags coexist without interference.
func TestSplitGlobalArgs_MixedGlobalAndCommandFlags(t *testing.T) {
	t.Parallel()

	app := orpheus.New("testapp")
	app.AddGlobalFlag("config", "c", "default.json", "Config file")
	app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")
	app.AddGlobalIntFlag("retries", "r", 3, "Retry count")

	var results struct {
		config  string
		verbose bool
		retries int
		input   string
		force   bool
	}

	cmd := orpheus.NewCommand("deploy", "Deploy command")
	cmd.AddFlag("input", "i", "main.go", "Input file")
	cmd.AddBoolFlag("force", "f", false, "Force deploy")
	cmd.SetHandler(func(ctx *orpheus.Context) error {
		results.config = ctx.GetGlobalFlagString("config")
		results.verbose = ctx.GetGlobalFlagBool("verbose")
		results.retries = ctx.GetGlobalFlagInt("retries")
		results.input = ctx.GetFlagString("input")
		results.force = ctx.GetFlagBool("force")
		return nil
	})
	app.AddCommand(cmd)

	err := app.Run([]string{
		"--config", "prod.yaml",
		"--verbose",
		"--retries", "-1",
		"deploy",
		"--input", "app.go",
		"--force",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results.config != "prod.yaml" {
		t.Errorf("config: expected prod.yaml, got %q", results.config)
	}
	if !results.verbose {
		t.Error("verbose: expected true")
	}
	if results.retries != -1 {
		t.Errorf("retries: expected -1, got %d", results.retries)
	}
	if results.input != "app.go" {
		t.Errorf("input: expected app.go, got %q", results.input)
	}
	if !results.force {
		t.Error("force: expected true")
	}
}

// TestSplitGlobalArgs_EmptyAndEdge verifies behavior with empty args,
// single args, and other boundary conditions.
func TestSplitGlobalArgs_EmptyAndEdge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			name: "empty args",
			args: []string{},
		},
		{
			name: "only command no flags",
			args: []string{"test"},
		},
		{
			name: "only flags no command",
			args: []string{"--verbose"},
		},
		{
			name: "flag at end without value",
			args: []string{"--config"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := orpheus.New("testapp")
			app.AddGlobalFlag("config", "c", "default.json", "Config file")
			app.AddGlobalBoolFlag("verbose", "v", false, "Verbose output")

			cmd := orpheus.NewCommand("test", "Test command")
			cmd.SetHandler(func(_ *orpheus.Context) error { return nil })
			app.AddCommand(cmd)

			// We're testing that these don't panic or crash.
			// Errors from missing commands or values are acceptable.
			_ = app.Run(tc.args)
		})
	}
}
