// subcommands_test.go: tests for subcommands functionality in Orpheus application framework
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
)

func TestSubcommandCreation(t *testing.T) {
	cmd := NewCommand("parent", "Parent command")
	subcmd := NewCommand("child", "Child command")

	cmd.AddSubcommand(subcmd)

	if !cmd.HasSubcommands() {
		t.Error("Expected command to have subcommands")
	}

	if subcmd.Parent() != cmd {
		t.Error("Expected subcommand to have correct parent")
	}

	subcommands := cmd.GetSubcommands()
	if len(subcommands) != 1 {
		t.Errorf("Expected 1 subcommand, got %d", len(subcommands))
	}

	if subcommands["child"] != subcmd {
		t.Error("Expected subcommand to be retrievable by name")
	}
}

func TestSubcommandFluent(t *testing.T) {
	var executed bool

	parent := NewCommand("parent", "Parent command")
	child := parent.Subcommand("child", "Child command", func(ctx *Context) error {
		executed = true
		return nil
	})

	// After the fix, Subcommand() returns the child, not the parent
	if child.Name() != "child" {
		t.Errorf("Expected child command name 'child', got '%s'", child.Name())
	}

	// Check that the parent has subcommands
	if !parent.HasSubcommands() {
		t.Error("Expected parent command to have subcommands")
	}

	subcmd := parent.GetSubcommand("child")
	if subcmd == nil {
		t.Fatal("Expected to find child subcommand")
	}

	// Test execution using the child command directly
	ctx := &Context{Args: []string{}}
	err := child.Execute(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !executed {
		t.Error("Expected subcommand handler to be executed")
	}
}

func TestSubcommandExecution(t *testing.T) {
	var executedCmd string
	var receivedArgs []string

	app := New("testapp")

	// Create parent command with subcommands
	remote := NewCommand("remote", "Manage remotes")

	remote.AddSubcommand(
		NewCommand("add", "Add remote").SetHandler(func(ctx *Context) error {
			executedCmd = "add"
			receivedArgs = make([]string, ctx.ArgCount())
			for i := 0; i < ctx.ArgCount(); i++ {
				receivedArgs[i] = ctx.GetArg(i)
			}
			return nil
		}),
	)

	remote.AddSubcommand(
		NewCommand("list", "List remotes").SetHandler(func(ctx *Context) error {
			executedCmd = "list"
			return nil
		}),
	)

	app.AddCommand(remote)

	// Test "remote add origin https://github.com/user/repo.git"
	err := app.Run([]string{"remote", "add", "origin", "https://github.com/user/repo.git"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if executedCmd != "add" {
		t.Errorf("Expected 'add' command to be executed, got '%s'", executedCmd)
	}

	expectedArgs := []string{"origin", "https://github.com/user/repo.git"}
	if len(receivedArgs) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(receivedArgs))
	}

	for i, expected := range expectedArgs {
		if i < len(receivedArgs) && receivedArgs[i] != expected {
			t.Errorf("Expected arg[%d] = '%s', got '%s'", i, expected, receivedArgs[i])
		}
	}

	// Test "remote list"
	executedCmd = ""
	err = app.Run([]string{"remote", "list"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if executedCmd != "list" {
		t.Errorf("Expected 'list' command to be executed, got '%s'", executedCmd)
	}
}

func TestSubcommandNotFound(t *testing.T) {
	app := New("testapp")

	remote := NewCommand("remote", "Manage remotes")
	remote.AddSubcommand(
		NewCommand("add", "Add remote").SetHandler(func(ctx *Context) error {
			return nil
		}),
	)

	app.AddCommand(remote)

	// Test non-existent subcommand
	err := app.Run([]string{"remote", "nonexistent"})
	if err == nil {
		t.Error("Expected error for non-existent subcommand")
	}
}

func TestSubcommandHelp(t *testing.T) {
	app := New("testapp")

	remote := NewCommand("remote", "Manage git remotes")
	remote.AddSubcommand(
		NewCommand("add", "Add a new remote").SetHandler(func(ctx *Context) error {
			return nil
		}),
	)
	remote.AddSubcommand(
		NewCommand("remove", "Remove a remote").SetHandler(func(ctx *Context) error {
			return nil
		}),
	)

	app.AddCommand(remote)

	// Test help generation for command with subcommands
	generator := NewHelpGenerator(app)
	help := generator.GenerateCommandHelp(remote)

	if !strings.Contains(help, "Available Subcommands:") {
		t.Error("Expected help to contain subcommands section")
	}

	if !strings.Contains(help, "add") {
		t.Error("Expected help to contain 'add' subcommand")
	}

	if !strings.Contains(help, "remove") {
		t.Error("Expected help to contain 'remove' subcommand")
	}

	if !strings.Contains(help, "Add a new remote") {
		t.Error("Expected help to contain subcommand description")
	}
}

func TestSubcommandWithFlags(t *testing.T) {
	var receivedFlag string

	app := New("testapp")

	db := NewCommand("db", "Database operations")
	migrate := NewCommand("migrate", "Run migrations").
		AddFlag("env", "e", "development", "Environment").
		SetHandler(func(ctx *Context) error {
			receivedFlag = ctx.GetFlagString("env")
			return nil
		})

	db.AddSubcommand(migrate)
	app.AddCommand(db)

	// Test subcommand with flags
	err := app.Run([]string{"db", "migrate", "--env", "production"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if receivedFlag != "production" {
		t.Errorf("Expected flag value 'production', got '%s'", receivedFlag)
	}
}

func TestSubcommandFullName(t *testing.T) {
	parent := NewCommand("parent", "Parent command")
	child := NewCommand("child", "Child command")
	grandchild := NewCommand("grandchild", "Grandchild command")

	parent.AddSubcommand(child)
	child.AddSubcommand(grandchild)

	if parent.FullName() != "parent" {
		t.Errorf("Expected 'parent', got '%s'", parent.FullName())
	}

	if child.FullName() != "parent child" {
		t.Errorf("Expected 'parent child', got '%s'", child.FullName())
	}

	if grandchild.FullName() != "parent child grandchild" {
		t.Errorf("Expected 'parent child grandchild', got '%s'", grandchild.FullName())
	}
}

func TestSubcommandWithoutHandler(t *testing.T) {
	app := New("testapp")

	// Command with subcommands but no handler should show help
	remote := NewCommand("remote", "Manage remotes")
	remote.AddSubcommand(
		NewCommand("add", "Add remote").SetHandler(func(ctx *Context) error {
			return nil
		}),
	)

	app.AddCommand(remote)

	// Running "remote" without subcommand should show help (no error)
	err := app.Run([]string{"remote"})
	if err != nil {
		t.Errorf("Expected help to be shown, got error: %v", err)
	}
}

func TestNestedSubcommands(t *testing.T) {
	var executedPath string

	app := New("testapp")

	// Create nested structure: config -> user -> set
	config := NewCommand("config", "Configuration management")
	user := NewCommand("user", "User configuration")
	set := NewCommand("set", "Set user value").SetHandler(func(ctx *Context) error {
		executedPath = "config user set"
		return nil
	})

	user.AddSubcommand(set)
	config.AddSubcommand(user)
	app.AddCommand(config)

	// Test nested execution: config user set
	err := app.Run([]string{"config", "user", "set"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if executedPath != "config user set" {
		t.Errorf("Expected 'config user set', got '%s'", executedPath)
	}
}

func TestSubcommandReturnsCreatedSubcommand(t *testing.T) {
	var flagValue string

	parent := NewCommand("parent", "Parent command")

	// Test that Subcommand() returns the created subcommand, not the parent
	child := parent.Subcommand("child", "Child command", func(ctx *Context) error {
		flagValue = ctx.GetFlagString("test-flag")
		return nil
	}).AddFlag("test-flag", "f", "default", "Test flag")

	// Verify that child is the subcommand, not the parent
	if child.Name() != "child" {
		t.Errorf("Expected subcommand name 'child', got '%s'", child.Name())
	}

	// Verify that the flag was added to the subcommand
	app := New("testapp")
	app.AddCommand(parent)

	err := app.Run([]string{"parent", "child", "--test-flag", "value"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if flagValue != "value" {
		t.Errorf("Expected flag value 'value', got '%s'", flagValue)
	}
}

// TestSubcommand_PrompterPropagatesToHandler is a regression guard
// for the bug where Context.Prompter() returned nil inside a
// subcommand handler even though the parent App had SetPrompter(). The
// fix was to copy ctx.prompter into the newCtx built inside
// handleSubcommands; without it, every interactive subcommand flow
// (e.g. `app cmd subcmd` calling AskSecret at the leaf) crashed
// with "no prompter configured" at runtime.
func TestSubcommand_PrompterPropagatesToHandler(t *testing.T) {
	app := New("propagation")
	// Custom prompter the test can recognise via type assertion.
	prompter := NewPrompterFrom(strings.NewReader("ignored\n"), &testWriter{}, -1)
	app.SetPrompter(prompter)

	parent := NewCommand("parent", "Parent command")
	var leafSawPrompter bool
	parent.AddSubcommand(
		NewCommand("child", "Leaf").SetHandler(func(ctx *Context) error {
			leafSawPrompter = ctx.Prompter() != nil
			return nil
		}),
	)
	app.AddCommand(parent)

	if err := app.Run([]string{"parent", "child"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !leafSawPrompter {
		t.Fatal("subcommand leaf observed nil prompter — App.SetPrompter must propagate through subcommand chain")
	}
}

// TestSubcommand_StoragePropagatesToHandler is the storage twin of
// the prompter regression test above. Same root cause, same fix.
func TestSubcommand_StoragePropagatesToHandler(t *testing.T) {
	app := New("propagation")
	app.SetStorage(NewMockStorage())

	parent := NewCommand("parent", "Parent command")
	var leafSawStorage bool
	parent.AddSubcommand(
		NewCommand("child", "Leaf").SetHandler(func(ctx *Context) error {
			leafSawStorage = ctx.Storage() != nil
			return nil
		}),
	)
	app.AddCommand(parent)

	if err := app.Run([]string{"parent", "child"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !leafSawStorage {
		t.Fatal("subcommand leaf observed nil storage — App.SetStorage must propagate through subcommand chain")
	}
}

// testWriter discards everything written to it. Local helper so the
// regression tests stay self-contained — no extra imports needed.
type testWriter struct{}

func (*testWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestSubcommand_PositionalAPIExcludesFlagTokens pins the new
// ctx.Positional / PositionalCount / GetPositional API: post-parse
// positional args MUST exclude flag tokens and their values. Before
// the API existed, handlers reaching for "the positional after my
// flag" had to filter ctx.Args (raw input) by hand against
// ctx.Flags — error-prone enough to silently swallow flag tokens as
// positionals (e.g. `app cmd --flag value` made ctx.GetArg(0)
// return "--flag"). ctx.ArgCount / GetArg keep their documented
// raw-input semantics; this test pins the new POSITIONAL contract
// alongside.
func TestSubcommand_PositionalAPIExcludesFlagTokens(t *testing.T) {
	app := New("argscope")
	parent := NewCommand("parent", "Parent")
	var leafPosCount int
	var leafFirstPos string
	var leafFlagValue string
	parent.AddSubcommand(
		NewCommand("child", "Leaf").
			AddFlag("token", "t", "", "auth token").
			SetHandler(func(ctx *Context) error {
				leafPosCount = ctx.PositionalCount()
				leafFirstPos = ctx.GetPositional(0)
				leafFlagValue = ctx.GetFlagString("token")
				return nil
			}),
	)
	app.AddCommand(parent)

	if err := app.Run([]string{"parent", "child", "--token", "secret-value"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if leafFlagValue != "secret-value" {
		t.Errorf("flag --token: got %q, want secret-value", leafFlagValue)
	}
	if leafPosCount != 0 {
		t.Errorf("PositionalCount after flag-only invocation: got %d, want 0", leafPosCount)
	}
	if leafFirstPos != "" {
		t.Errorf("GetPositional(0) MUST be empty when no positional supplied; got %q", leafFirstPos)
	}

	// Positional alongside a flag: PositionalCount returns exactly
	// the non-flag tokens, in order.
	leafPosCount = 0
	leafFirstPos = ""
	leafFlagValue = ""
	if err := app.Run([]string{"parent", "child", "--token", "v", "real-positional"}); err != nil {
		t.Fatalf("Run with positional: %v", err)
	}
	if leafPosCount != 1 || leafFirstPos != "real-positional" {
		t.Errorf("positional after flag: got count=%d first=%q, want count=1 first=real-positional",
			leafPosCount, leafFirstPos)
	}
	if leafFlagValue != "v" {
		t.Errorf("flag value lost when positional follows: got %q, want v", leafFlagValue)
	}
}

// TestContext_PositionalAPI_ZeroValueSafety pins the nil-Flags
// contract documented on the Positional* methods — handlers that
// hold a Context built outside the dispatch chain (tests, fakes)
// must not panic when calling the new accessors.
func TestContext_PositionalAPI_ZeroValueSafety(t *testing.T) {
	ctx := &Context{} // no Flags wired
	if got := ctx.Positional(); got != nil {
		t.Errorf("Positional() with nil Flags: got %v, want nil", got)
	}
	if got := ctx.PositionalCount(); got != 0 {
		t.Errorf("PositionalCount() with nil Flags: got %d, want 0", got)
	}
	if got := ctx.GetPositional(0); got != "" {
		t.Errorf("GetPositional(0) with nil Flags: got %q, want empty", got)
	}
}
