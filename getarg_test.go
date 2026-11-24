// getarg_test.go: GetArg and ArgCount read positionals; not-found errors name
// the command once
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
)

// A flag before the operands is not an operand: "remote add -f origin URL"
// added a remote named "-f" in the gitlike example.
func TestGetArgSkipsFlags(t *testing.T) {
	var got []string
	var count int
	remote := NewCommand("remote", "Remotes")
	remote.AddSubcommand(
		NewCommand("add", "Add").
			AddBoolFlag("fetch", "f", false, "fetch").
			AddFlag("tags", "t", "", "tags").
			SetHandler(func(ctx *Context) error {
				count = ctx.ArgCount()
				got = []string{ctx.GetArg(0), ctx.GetArg(1), ctx.GetArg(2)}
				return nil
			}),
	)
	app := New("app")
	app.AddCommand(remote)

	for _, args := range [][]string{
		{"remote", "add", "-f", "origin", "URL"},
		{"remote", "add", "--tags", "all", "origin", "URL"},
		{"remote", "add", "origin", "-f", "URL"},
		{"remote", "add", "origin", "URL", "--fetch"},
	} {
		if err := app.Run(args); err != nil {
			t.Fatalf("Run(%q): %v", args, err)
		}
		if count != 2 || got[0] != "origin" || got[1] != "URL" || got[2] != "" {
			t.Errorf("Run(%q): ArgCount=%d GetArg=%q, want 2 [origin URL \"\"]", args, count, got)
		}
	}
}

// A Context built outside the dispatch path has no parsed flags: GetArg reads
// ctx.Args, the only view there is.
func TestGetArgWithoutParsedFlags(t *testing.T) {
	ctx := &Context{Args: []string{"a", "b"}}
	if ctx.ArgCount() != 2 || ctx.GetArg(1) != "b" || ctx.GetArg(2) != "" || ctx.GetArg(-1) != "" {
		t.Errorf("ArgCount=%d GetArg(1)=%q, want 2 and b", ctx.ArgCount(), ctx.GetArg(1))
	}
}

// The command appears once in Error(): it used to read
// "command 'stauts': [ORF1002]: command 'stauts' not found".
func TestNotFoundNamesCommandOnce(t *testing.T) {
	app := New("app")
	app.AddCommand(NewCommand("remote", "Remotes").
		AddSubcommand(NewCommand("add", "Add").SetHandler(func(*Context) error { return nil })))

	for args, want := range map[string]string{
		"stauts":    "command 'stauts': [ORF1002]: unknown command",
		"remote ad": "command 'remote': [ORF1002]: unknown subcommand 'ad'",
	} {
		err := app.Run(strings.Fields(args))
		if err == nil || err.Error() != want {
			t.Errorf("Run(%q) = %v, want %q", args, err, want)
		}
	}
	if err := app.Run([]string{"help", "stauts"}); err == nil || err.Error() != "command 'stauts': [ORF1002]: unknown command" {
		t.Errorf("help stauts = %v", err)
	}
}
