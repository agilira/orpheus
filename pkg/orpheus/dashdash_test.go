// dashdash_test.go: "--" ends option parsing for the help scan too, so a command
// that forwards arguments to another program receives -h and --help untouched.
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

func TestHelpFlagAfterDoubleDashIsForwarded(t *testing.T) {
	var got []string
	app := New("wrap")
	app.AddCommand(NewCommand("run", "Run a program").SetHandler(func(ctx *Context) error {
		got = ctx.Positional()
		return nil
	}))
	if err := app.RunContext(context.Background(), []string{"run", "--", "tool", "-h", "--help"}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if strings.Join(got, " ") != "tool -h --help" {
		t.Fatalf("handler received %q, want the arguments after --", got)
	}
}

func TestHelpFlagBeforeDoubleDashStillShowsHelp(t *testing.T) {
	called := false
	cmd := NewCommand("run", "Run a program").SetHandler(func(*Context) error {
		called = true
		return nil
	})
	if !cmd.hasHelpFlag([]string{"-h", "--", "tool"}) || !cmd.hasHelpFlag([]string{"--help"}) {
		t.Fatal("help flag before -- not recognized")
	}
	if cmd.hasHelpFlag([]string{"--", "-h"}) || cmd.hasHelpFlag([]string{"x", "--", "--help"}) {
		t.Fatal("help flag after -- treated as ours")
	}
	if called {
		t.Fatal("handler ran during the scan")
	}
}

func TestHelpOmitsEmptyDefaults(t *testing.T) {
	app := New("app")
	app.AddGlobalFlag("token", "", "", "API token")
	app.AddGlobalFlag("region", "", "eu", "region")
	cmd := NewCommand("go", "Do it").SetHandler(func(*Context) error { return nil }).
		AddFlag("name", "", "", "a name").
		AddStringSliceFlag("tags", "", nil, "tags").
		AddIntFlag("count", "", 0, "count")
	app.AddCommand(cmd)

	help := app.GenerateHelp() + NewHelpGenerator(app).GenerateCommandHelp(cmd)
	if strings.Contains(help, "(default: )") || strings.Contains(help, "(default: [])") {
		t.Fatalf("help shows an empty default:\n%s", help)
	}
	for _, want := range []string{"(default: eu)", "(default: 0)"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help lacks %q:\n%s", want, help)
		}
	}
}
