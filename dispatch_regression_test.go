// dispatch_regression_test.go: regression tests and benchmarks for command
// dispatch, built-in help/version handling, help rendering and completion
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/agilira/orpheus/errs"
	"github.com/agilira/orpheus/storage"
)

// dispatchCapture runs fn with os.Stdout redirected and returns what it printed.
func dispatchCapture(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()
	defer func() { os.Stdout = old }()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

// 1. "-h" after a subcommand prints the PARENT's help, not the subcommand's.
func TestDispatch_SubcommandHelpShowsParentHelp(t *testing.T) {
	app := New("app")
	remote := NewCommand("remote", "Manage remotes")
	remote.Subcommand("add", "Add a remote", func(*Context) error { return nil }).
		AddBoolFlag("force", "f", false, "Force add")
	app.AddCommand(remote)

	out := dispatchCapture(t, func() { _ = app.Run([]string{"remote", "add", "-h"}) })
	if !strings.Contains(out, "Add a remote") || !strings.Contains(out, "--force") {
		t.Fatalf("`app remote add -h` did not show the help of 'add':\n%s", out)
	}
}

// 2. A command that registers its own -h (host) never gets it: Execute prints help.
func TestDispatch_CommandShortHClaimIgnored(t *testing.T) {
	app := New("app")
	var host string
	app.AddCommand(NewCommand("connect", "Connect").SetHandler(func(c *Context) error {
		host = c.GetFlagString("host")
		return nil
	}).AddFlag("host", "h", "localhost", "Host"))

	_ = dispatchCapture(t, func() { _ = app.Run([]string{"connect", "-h", "example.com"}) })
	if host != "example.com" {
		t.Fatalf("registered -h was intercepted as help; handler saw host=%q", host)
	}
}

// 3. A flag VALUE equal to "-h"/"--help" triggers help instead of being the value.
func TestDispatch_FlagValueHelpTriggersHelp(t *testing.T) {
	app := New("app")
	var pat string
	app.AddCommand(NewCommand("grep", "Search").SetHandler(func(c *Context) error {
		pat = c.GetFlagString("pattern")
		return nil
	}).AddFlag("pattern", "p", "", "Pattern"))

	_ = dispatchCapture(t, func() { _ = app.Run([]string{"grep", "--pattern", "-h"}) })
	if pat != "-h" {
		t.Fatalf("value '-h' of --pattern became a help request; pattern=%q", pat)
	}
}

// 4. --help / --version after a global flag: flash-flags' own help + error, or unknown flag.
func TestDispatch_HelpAfterGlobalFlagErrors(t *testing.T) {
	app := New("app").SetVersion("1.0.0")
	app.AddGlobalBoolFlag("verbose", "", false, "Verbose")
	app.Command("x", "x", func(*Context) error { return nil })

	var errHelp, errVer error
	_ = dispatchCapture(t, func() {
		errHelp = app.Run([]string{"--verbose", "--help"})
		errVer = app.Run([]string{"--verbose", "--version"})
	})
	if errHelp != nil || errVer != nil {
		t.Fatalf("`--verbose --help` err=%v; `--verbose --version` err=%v", errHelp, errVer)
	}
}

// 5. Combined short boolean global flags swallow the command name.
func TestDispatch_CombinedShortGlobalSwallowsCommand(t *testing.T) {
	app := New("app")
	app.AddGlobalBoolFlag("verbose", "x", false, "Verbose")
	app.AddGlobalBoolFlag("quiet", "q", false, "Quiet")
	ran := false
	app.Command("deploy", "Deploy", func(*Context) error { ran = true; return nil })

	_ = dispatchCapture(t, func() { _ = app.Run([]string{"-xq", "deploy"}) })
	if !ran {
		t.Fatal("`app -xq deploy` did not run deploy (command consumed as flag value)")
	}
}

// 6. Repeated Run on the same App leaks flag values and Changed between runs.
func TestDispatch_RepeatedRunLeaksFlags(t *testing.T) {
	app := New("app")
	app.AddGlobalBoolFlag("verbose", "", false, "Verbose")
	var name string
	var gChanged, changed bool
	app.AddCommand(NewCommand("greet", "Greet").SetHandler(func(c *Context) error {
		name = c.GetFlagString("name")
		changed = c.FlagChanged("name")
		gChanged = c.GlobalFlagChanged("verbose")
		return nil
	}).AddFlag("name", "n", "world", "Name"))

	if err := app.Run([]string{"--verbose", "greet", "--name", "bob"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"greet"}); err != nil {
		t.Fatal(err)
	}
	if name != "world" || changed || gChanged {
		t.Fatalf("second run saw state of first: name=%q changed=%v globalChanged=%v", name, changed, gChanged)
	}
}

// 7. Help "(default: X)" prints the current parsed value, not the default.
func TestDispatch_HelpDefaultShowsParsedValue(t *testing.T) {
	app := New("app")
	app.AddGlobalFlag("config", "", "app.yaml", "Config file")
	app.Command("build", "x", func(*Context) error { return nil })

	out := dispatchCapture(t, func() { _ = app.Run([]string{"--config", "secret.yaml", "help"}) })
	if !strings.Contains(out, "(default: app.yaml)") {
		t.Fatalf("help shows parsed value as default:\n%s", out)
	}
}

// 8. Usage line of a nested subcommand omits the parent path.
func TestDispatch_SubcommandUsageMissingParent(t *testing.T) {
	app := New("app")
	remote := NewCommand("remote", "Manage remotes")
	add := remote.Subcommand("add", "Add a remote", func(*Context) error { return nil })
	app.AddCommand(remote)

	out := NewHelpGenerator(app).GenerateCommandHelp(add)
	if !strings.Contains(out, "Usage: app remote add") {
		t.Fatalf("usage line lacks parent path:\n%s", out)
	}
}

// 9. `app help remote add` ignores "add" and shows remote's help.
func TestDispatch_HelpCommandNested(t *testing.T) {
	app := New("app")
	remote := NewCommand("remote", "Manage remotes")
	remote.Subcommand("add", "Add a remote", func(*Context) error { return nil }).
		AddBoolFlag("force", "f", false, "Force add")
	app.AddCommand(remote)

	out := dispatchCapture(t, func() { _ = app.Run([]string{"help", "remote", "add"}) })
	if !strings.Contains(out, "--force") {
		t.Fatalf("`app help remote add` did not show add's help:\n%s", out)
	}
}

// 10. Generated zsh script: every command must reach _describe as ONE
// "name:description" word, apostrophes and colons included.
func TestDispatch_ZshDescribeQuoting(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	app := New("app")
	app.Command("start", "Start the service", func(*Context) error { return nil })
	app.Command("cfg", "Show user's config", func(*Context) error { return nil })
	script := app.GenerateCompletion("zsh")

	i := strings.Index(script, "topics=(")
	j := strings.Index(script, "\n\n    _arguments")
	if i < 0 || j < i {
		t.Fatalf("no _describe arrays in the zsh script:\n%s", script)
	}
	// POSIX single-quote rules are identical in zsh and bash.
	stmt := script[i:j] + "\nprintf '[%s]\\n' \"${commands[@]}\""
	out, err := exec.Command("bash", "-c", stmt).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	want := "[cfg:Show user's config]\n[start:Start the service]\n[help:Show help for commands]\n"
	if string(out) != want {
		t.Fatalf("_describe specs:\n%s\nwant:\n%s", out, want)
	}
	if !strings.Contains(script, "_describe 'commands' commands") {
		t.Fatal("_describe is not given the array name")
	}
}

// 11. App help panics when every command name is shorter than "help" minus 1 (len 1).
func TestDispatch_AppHelpPanicsShortNames(t *testing.T) {
	app := New("app")
	app.Command("x", "Do x", func(*Context) error { return nil })
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("`app` (no args) panicked rendering help: %v", r)
		}
	}()
	_ = dispatchCapture(t, func() { _ = app.Run(nil) })
}

// benchDispatchApp builds an application with global flags and a nested
// subcommand, the shape the dispatch hot path walks most.
func benchDispatchApp() *App {
	app := New("bench").SetVersion("1.0.0")
	app.AddGlobalBoolFlag("verbose", "x", false, "Verbose")
	app.AddGlobalFlag("config", "c", "app.yaml", "Config file")
	deploy := NewCommand("deploy", "Deploy application").SetHandler(func(c *Context) error {
		_ = c.GetFlagString("env")
		return nil
	})
	deploy.AddFlag("env", "e", "prod", "Environment")
	deploy.AddBoolFlag("force", "f", false, "Force")
	deploy.AddFlag("timeout", "t", "30", "Timeout")
	app.AddCommand(deploy)
	remote := NewCommand("remote", "Manage remotes")
	remote.Subcommand("add", "Add a remote", func(*Context) error { return nil }).
		AddBoolFlag("force", "f", false, "Force add")
	app.AddCommand(remote)
	return app
}

func BenchmarkDispatchCommandFlags(b *testing.B) {
	app := benchDispatchApp()
	args := []string{"deploy", "--env", "staging", "--force", "--timeout", "60"}
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := app.RunContext(ctx, args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchGlobalAndSubcommand(b *testing.B) {
	app := benchDispatchApp()
	args := []string{"-x", "--config", "c.yaml", "remote", "add", "-f", "origin", "url"}
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := app.RunContext(ctx, args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatchAppHelp(b *testing.B) {
	app := benchDispatchApp()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = app.GenerateHelp()
	}
}

func BenchmarkDispatchCommandHelp(b *testing.B) {
	app := benchDispatchApp()
	gen := NewHelpGenerator(app)
	cmd := app.commands["deploy"]
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = gen.GenerateCommandHelp(cmd)
	}
}

// A cluster whose last key takes a value consumes the next argument, and a
// help spelling consumed as a global flag's value stays a value.
func TestDispatch_GlobalClusterValueAndHelpAsValue(t *testing.T) {
	app := New("app")
	app.AddGlobalBoolFlag("verbose", "x", false, "Verbose")
	app.AddGlobalFlag("config", "c", "app.yaml", "Config")
	var cfg string
	app.Command("deploy", "Deploy", func(c *Context) error {
		cfg = c.GetGlobalFlagString("config")
		return nil
	})
	for _, args := range [][]string{{"-xc", "-h", "deploy"}, {"--config", "-h", "deploy"}} {
		cfg = ""
		_ = dispatchCapture(t, func() {
			if err := app.Run(args); err != nil {
				t.Errorf("%q: %v", args, err)
			}
		})
		if cfg != "-h" {
			t.Errorf("%q: config=%q, want -h", args, cfg)
		}
	}
}

// `app help remote bogus` reports the unknown subcommand.
func TestDispatch_HelpCommandUnknownSubcommand(t *testing.T) {
	app := New("app")
	remote := NewCommand("remote", "Manage remotes")
	remote.Subcommand("add", "Add a remote", func(*Context) error { return nil })
	app.AddCommand(remote)
	err := app.Run([]string{"help", "remote", "bogus"})
	if e, ok := err.(*errs.Error); !ok || !e.IsNotFoundError() {
		t.Fatalf("err = %v, want a not-found error", err)
	}
}

// The help scan stops at "--" once a claimed -h has consumed its value, and
// skips positionals: neither the "--help" after "--" nor "pos" is help.
func TestDispatch_HelpScanStopsAtDoubleDash(t *testing.T) {
	var host string
	var rest []string
	cmd := NewCommand("connect", "Connect").SetHandler(func(c *Context) error {
		host = c.GetFlagString("host")
		rest = c.Positional()
		return nil
	}).AddFlag("host", "h", "localhost", "Host")
	app := New("app").AddCommand(cmd)

	if err := app.Run([]string{"connect", "pos", "-h", "example.com", "--", "--help"}); err != nil {
		t.Fatal(err)
	}
	if host != "example.com" || strings.Join(rest, " ") != "pos --help" {
		t.Fatalf("host=%q positional=%q", host, rest)
	}
}

// A flag registered on Flags() directly has no recorded default: help shows
// its value, the only one known.
func TestDispatch_HelpDefaultOfUnrecordedFlag(t *testing.T) {
	app := New("app")
	cmd := NewCommand("build", "Build")
	cmd.Flags().String("target", "linux", "Target")
	app.AddCommand(cmd)

	out := NewHelpGenerator(app).GenerateCommandHelp(cmd)
	if !strings.Contains(out, "(default: linux)") {
		t.Fatalf("help lacks the default of a directly registered flag:\n%s", out)
	}
}

// An App without a global flag set registers no short keys: only the
// built-ins -v and -h are boolean.
func TestDispatch_ShortKindWithoutGlobalFlags(t *testing.T) {
	app := &App{}
	if !app.isShortBooleanFlag("-v") || app.isShortBooleanFlag("-x") {
		t.Fatal("short flag kind wrong without a global flag set")
	}
}

// Run(nil) running the default command starts from fresh global flags, as
// any other run does.
func TestDispatch_DefaultCommandResetsGlobals(t *testing.T) {
	app := New("app")
	app.AddGlobalBoolFlag("verbose", "", false, "Verbose")
	var verbose, changed bool
	app.Command("greet", "Greet", func(c *Context) error {
		verbose = c.GetGlobalFlagBool("verbose")
		changed = c.GlobalFlagChanged("verbose")
		return nil
	})
	app.SetDefaultCommand("greet")

	if err := app.Run([]string{"--verbose", "greet"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(nil); err != nil {
		t.Fatal(err)
	}
	if verbose || changed {
		t.Fatalf("default command saw the previous run's globals: verbose=%v changed=%v", verbose, changed)
	}
}

// `app help <TAB>` in zsh offers the commands help can describe, and not help
// itself, matching the bash and fish scripts.
func TestDispatch_ZshHelpTopicsExcludeHelp(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	app := New("app")
	app.Command("start", "Start the service", func(*Context) error { return nil })
	script := app.GenerateCompletion("zsh")

	i := strings.Index(script, "topics=(")
	j := strings.Index(script, "\n\n    _arguments")
	if i < 0 || j < i || !strings.Contains(script, "_describe 'commands' topics") {
		t.Fatalf("no help topics array in the zsh script:\n%s", script)
	}
	stmt := script[i:j] + "\nprintf '[%s]' \"${topics[@]}\""
	out, err := exec.Command("bash", "-c", stmt).CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	if string(out) != "[start:Start the service]" {
		t.Fatalf("help topics = %s", out)
	}
}

// "--" consumed as a flag's value does not end the help scan.
func TestDispatch_HelpAfterDoubleDashValue(t *testing.T) {
	app := New("app")
	ran := false
	app.AddCommand(NewCommand("grep", "Search").SetHandler(func(*Context) error {
		ran = true
		return nil
	}).AddFlag("pattern", "p", "", "Pattern"))

	var err error
	out := dispatchCapture(t, func() { err = app.Run([]string{"grep", "--pattern", "--", "-h"}) })
	if err != nil || ran || !strings.Contains(out, "Usage: app grep") {
		t.Fatalf("err=%v ran=%v output:\n%s", err, ran, out)
	}
}

// The reason a storage plugin failed to load stays in the error chain.
func TestDispatch_StorageErrorKeepsCause(t *testing.T) {
	app := New("app").ConfigureStorage(&storage.Config{Provider: "no-such-provider"})
	for err := app.StorageError(); err != nil; err = errors.Unwrap(err) {
		if e, ok := err.(*errs.Error); ok && e.ErrorCode() == errs.ErrCodePluginError {
			return
		}
	}
	t.Fatalf("plugin load cause lost: %v", app.StorageError())
}
