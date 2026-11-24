package orpheus

import (
	"strings"
	"testing"
)

func TestCompletionScriptsAreDeterministic(t *testing.T) {
	newApp := func() *App {
		app := New("myapp").SetVersion("1.0.0")
		for _, n := range []string{"build", "deploy", "test", "clean", "status", "init", "run", "watch"} {
			name := n
			app.Command(name, "does "+name, func(ctx *Context) error { return nil })
		}
		return app
	}

	for _, shell := range []string{"bash", "zsh", "fish"} {
		first := newApp().GenerateCompletion(shell)
		for i := 0; i < 20; i++ {
			if got := newApp().GenerateCompletion(shell); got != first {
				t.Fatalf("%s completion is not deterministic (run %d differs)", shell, i)
			}
		}
	}
}

func TestCompletionQuotesDescriptionsSafely(t *testing.T) {
	app := New("myapp")
	app.Command("config", "Show user's config", func(ctx *Context) error { return nil })

	for _, shell := range []string{"zsh", "fish"} {
		script := app.GenerateCompletion(shell)
		if strings.Contains(script, "user's config") {
			t.Errorf("%s: an apostrophe in a description escapes its single-quoted string:\n%s", shell, script)
		}
	}
}

// A registered shorthand must be discoverable by completion, and every
// spelling must appear exactly once even when the application claims one the
// framework also offers.
func TestCompleteFlagsSuggestsShorthandsWithoutDuplicates(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalBoolFlag("verbose", "v", false, "verbose output")
	cmd := NewCommand("build", "build it").
		AddFlag("host", "H", "", "target host").
		SetHandler(func(ctx *Context) error { return nil })
	app.AddCommand(cmd)

	got := app.Complete([]string{"build", "-"}, 2).Suggestions

	seen := map[string]int{}
	for _, s := range got {
		seen[s]++
	}
	for s, n := range seen {
		if n > 1 {
			t.Errorf("%q suggested %d times: %q", s, n, got)
		}
	}
	for _, want := range []string{"-H", "--host", "-v", "--verbose", "--help", "--version"} {
		if seen[want] == 0 {
			t.Errorf("%q is accepted by the parser but not suggested: %q", want, got)
		}
	}
}

// The generated fish script must not offer a built-in spelling the
// application has taken over.
func TestFishScriptDropsClaimedBuiltinSpellings(t *testing.T) {
	app := New("myapp").SetVersion("1.0.0")
	app.AddGlobalFlag("version", "v", "", "version to deploy")

	script := app.GenerateCompletion("fish")

	if strings.Contains(script, "Show version") {
		t.Errorf("fish script still offers the built-in version flag although both spellings are claimed:\n%s", script)
	}
	if !strings.Contains(script, "-s h -l help") {
		t.Errorf("fish script dropped the built-in help flag, which is unclaimed:\n%s", script)
	}
}
