// builtin_flags_test.go: the built-in --help/--version flags must never
// shadow a flag the application registered under the same name or short key.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import "testing"

// A user-registered -v must reach the parser, not the version printer.
func TestUserShortVFlagIsNotHijackedByVersion(t *testing.T) {
	ran := false
	var verbose bool

	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalBoolFlag("verbose", "v", false, "verbose output")
	app.Command("build", "build it", func(ctx *Context) error {
		ran = true
		verbose = ctx.GetGlobalFlagBool("verbose")
		return nil
	})

	if err := app.Run([]string{"-v", "build"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ran {
		t.Fatal("build never ran: -v was consumed by the built-in version flag")
	}
	if !verbose {
		t.Fatal("--verbose was not set: -v did not reach the flag parser")
	}
}

// Same for a user-registered -h.
func TestUserShortHFlagIsNotHijackedByHelp(t *testing.T) {
	ran := false
	var host string

	app := New("myapp")
	app.AddGlobalFlag("host", "h", "", "target host")
	app.Command("deploy", "deploy it", func(ctx *Context) error {
		ran = true
		host = ctx.GetGlobalFlagString("host")
		return nil
	})

	if err := app.Run([]string{"-h", "example.com", "deploy"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ran {
		t.Fatal("deploy never ran: -h was consumed by the built-in help flag")
	}
	if host != "example.com" {
		t.Fatalf("host = %q, want %q: -h did not reach the flag parser", host, "example.com")
	}
}

// The long forms defer too, when the application claims the name.
func TestUserLongVersionFlagIsNotHijacked(t *testing.T) {
	ran := false
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalFlag("version", "", "", "version to deploy")
	app.Command("deploy", "deploy it", func(ctx *Context) error { ran = true; return nil })

	if err := app.Run([]string{"--version", "2.0.0", "deploy"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ran {
		t.Fatal("deploy never ran: --version was consumed by the built-in version printer")
	}
}

// With nothing registered, the built-ins keep working.
func TestBuiltinFlagsStillWorkWhenUnclaimed(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "--version", "-v"} {
		app := New("myapp").SetVersion("1.0.0")
		app.Command("build", "build it", func(ctx *Context) error {
			t.Errorf("%s: build ran, but the built-in flag should have handled it", arg)
			return nil
		})
		if err := app.Run([]string{arg, "build"}); err != nil {
			t.Fatalf("%s: %v", arg, err)
		}
	}
}
