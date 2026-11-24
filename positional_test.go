// positional_test.go: ctx.Args and ctx.Positional() must describe the same
// arguments -- the parser's view differs only by flag tokens, never by a
// positional that happens to spell the command's own name.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"reflect"
	"testing"
)

// "myapp copy copy dest" must reach the handler with both positionals.
func TestPositionalSpellingTheCommandNameSurvives(t *testing.T) {
	var raw, pos []string

	app := New("myapp")
	app.Command("copy", "copy a file", func(ctx *Context) error {
		raw, pos = ctx.Args, ctx.Positional()
		return nil
	})

	if err := app.Run([]string{"copy", "copy", "dest"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"copy", "dest"}
	if !reflect.DeepEqual(raw, want) {
		t.Errorf("ctx.Args = %q, want %q", raw, want)
	}
	if !reflect.DeepEqual(pos, want) {
		t.Errorf("ctx.Positional() = %q, want %q", pos, want)
	}
}

// Same at a subcommand boundary: "git remote add remote url".
func TestSubcommandPositionalSpellingItsOwnNameSurvives(t *testing.T) {
	var pos []string

	root := NewCommand("remote", "remote ops")
	root.Subcommand("add", "add a remote", func(ctx *Context) error {
		pos = ctx.Positional()
		return nil
	})

	app := New("git")
	app.AddCommand(root)

	if err := app.Run([]string{"remote", "add", "add", "https://example.com"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"add", "https://example.com"}
	if !reflect.DeepEqual(pos, want) {
		t.Fatalf("ctx.Positional() = %q, want %q", pos, want)
	}
}

// Flags are still excluded from the positional view, and still parsed.
func TestPositionalExcludesFlagsButKeepsValues(t *testing.T) {
	var pos []string
	var force bool

	cmd := NewCommand("copy", "copy a file").
		AddBoolFlag("force", "f", false, "overwrite").
		SetHandler(func(ctx *Context) error {
			pos = ctx.Positional()
			force = ctx.GetFlagBool("force")
			return nil
		})

	app := New("myapp")
	app.AddCommand(cmd)

	if err := app.Run([]string{"copy", "src", "--force", "copy"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !force {
		t.Error("--force was not parsed")
	}
	want := []string{"src", "copy"}
	if !reflect.DeepEqual(pos, want) {
		t.Fatalf("ctx.Positional() = %q, want %q", pos, want)
	}
}
