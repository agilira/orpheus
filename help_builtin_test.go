// help_builtin_test.go: the help text must describe what the parser will
// actually do with -h and -v, not what the framework would do unclaimed.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
)

func TestHelpDoesNotAdvertiseClaimedShortKeys(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalBoolFlag("verbose", "v", false, "verbose output")
	app.AddGlobalFlag("host", "h", "", "target host")

	help := app.GenerateHelp()

	if strings.Contains(help, "-v, --version") {
		t.Errorf("help advertises -v for --version although the application claims -v:\n%s", help)
	}
	if strings.Contains(help, "-h, --help") {
		t.Errorf("help advertises -h for --help although the application claims -h:\n%s", help)
	}
	// The long forms still work, so they are still listed.
	if !strings.Contains(help, "--version") || !strings.Contains(help, "--help") {
		t.Errorf("the long built-in forms disappeared from help:\n%s", help)
	}
	// And the application's own flags are described.
	if !strings.Contains(help, "verbose output") || !strings.Contains(help, "target host") {
		t.Errorf("application flags missing from help:\n%s", help)
	}
}

// Claiming only the long name leaves the short key to the built-in, and help
// says so: -v alone, never "-v, --version".
func TestHelpListsOnlyTheSpellingsStillFree(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalFlag("version", "V", "", "version to deploy")

	help := app.GenerateHelp()

	if strings.Contains(help, "-v, --version") {
		t.Errorf("help advertises --version for the built-in although it is claimed:\n%s", help)
	}
	if !strings.Contains(help, "  -v ") {
		t.Errorf("help dropped -v, which still reaches the built-in version flag:\n%s", help)
	}
	if !strings.Contains(help, "version to deploy") {
		t.Errorf("the application's own --version is missing from help:\n%s", help)
	}
}

// Claiming every spelling removes the built-in line altogether.
func TestHelpDropsBuiltinEntirelyWhenEverySpellingIsClaimed(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalFlag("version", "v", "", "version to deploy")

	help := app.GenerateHelp()

	if strings.Contains(help, "Show version") {
		t.Errorf("help still offers the built-in version flag although both spellings are claimed:\n%s", help)
	}
	if !strings.Contains(help, "version to deploy") {
		t.Errorf("the application's own --version is missing from help:\n%s", help)
	}
}

func TestCommandHelpDoesNotAdvertiseClaimedShortKey(t *testing.T) {
	cmd := NewCommand("deploy", "deploy it").
		AddFlag("host", "h", "", "target host").
		SetHandler(func(ctx *Context) error { return nil })

	app := New("myapp")
	app.AddCommand(cmd)

	help := NewHelpGenerator(app).GenerateCommandHelp(cmd)

	// Only the command's own section is at stake: the global section belongs
	// to the app's flag set, which has claimed nothing.
	cmdSection, _, _ := strings.Cut(help, "Global Flags:")

	if strings.Contains(cmdSection, "-h, --help") {
		t.Errorf("command help advertises -h for --help although the command claims -h:\n%s", help)
	}
	if !strings.Contains(cmdSection, "--help") {
		t.Errorf("--help disappeared from command help:\n%s", help)
	}
	if !strings.Contains(cmdSection, "-h, --host") {
		t.Errorf("the command's own -h shorthand is missing from help:\n%s", help)
	}
}

// Unclaimed, everything is advertised as before.
func TestHelpAdvertisesBuiltinsWhenUnclaimed(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.Command("build", "build it", func(ctx *Context) error { return nil })

	help := app.GenerateHelp()
	for _, want := range []string{"-h, --help", "-v, --version"} {
		if !strings.Contains(help, want) {
			t.Errorf("help is missing %q:\n%s", want, help)
		}
	}
}
