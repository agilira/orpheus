package orpheus

import (
	"context"
	"testing"
)

// A subcommand handler must observe the same context the App was run with.
func TestSubcommandInheritsRunContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: the handler must see it

	var seen error
	root := NewCommand("remote", "remote ops")
	root.Subcommand("add", "add a remote", func(ctx *Context) error {
		seen = ctx.Context().Err()
		return nil
	})

	app := New("git")
	app.AddCommand(root)

	if err := app.RunContext(parent, []string{"remote", "add", "origin"}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if seen == nil {
		t.Fatal("subcommand handler saw a live context; the cancelled parent context was not propagated")
	}
}

// Same check one level deeper.
func TestNestedSubcommandInheritsRunContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	var seen error
	root := NewCommand("a", "a")
	mid := NewCommand("b", "b")
	mid.Subcommand("c", "c", func(ctx *Context) error {
		seen = ctx.Context().Err()
		return nil
	})
	root.AddSubcommand(mid)

	app := New("app")
	app.AddCommand(root)

	if err := app.RunContext(parent, []string{"a", "b", "c"}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if seen == nil {
		t.Fatal("nested handler saw a live context; parent context not propagated")
	}
}
