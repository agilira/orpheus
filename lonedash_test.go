// lonedash_test.go: a lone "-" reaches the handler as an operand, as
// POSIX tools expect ("prog add job -" reads standard input).
//
// Copyright (c) 2025-2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"strings"
	"testing"
)

func TestLoneDashReachesTheHandler(t *testing.T) {
	for name, args := range map[string][]string{
		"after a flag":     {"add", "job", "--when", "daily", "-"},
		"before a flag":    {"add", "job", "-", "--when", "daily"},
		"in a subcommand":  {"schedule", "add", "job", "--when", "daily", "-"},
		"with global flag": {"--verbose", "add", "job", "--when", "daily", "-"},
	} {
		var got []string
		var when string
		handler := func(ctx *Context) error {
			got, when = ctx.Positional(), ctx.GetFlagString("when")
			return nil
		}
		app := New("prog").AddGlobalBoolFlag("verbose", "", false, "talk more")
		app.AddCommand(NewCommand("add", "Add").SetHandler(handler).AddFlag("when", "", "", "when"))
		app.AddCommand(NewCommand("schedule", "Jobs").SetHandler(func(*Context) error { return nil }).
			AddSubcommand(NewCommand("add", "Add").SetHandler(handler).AddFlag("when", "", "", "when")))
		if err := app.RunContext(context.Background(), args); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Join(got, " ") != "job -" || when != "daily" {
			t.Errorf("%s: positional %q, when %q", name, got, when)
		}
	}
}

// A lone "-" before the command is not swallowed as a global flag: it is
// taken for the command, and reported as an unknown one.
func TestLoneDashIsNotAGlobalFlag(t *testing.T) {
	app := New("prog")
	app.AddCommand(NewCommand("add", "Add").SetHandler(func(*Context) error { return nil }))
	global, rest, _ := app.splitGlobalArgs([]string{"-", "add"})
	if len(global) != 0 || strings.Join(rest, " ") != "- add" {
		t.Fatalf("global %q, rest %q", global, rest)
	}
}
