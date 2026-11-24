// prompt_wiring_test.go: the Prompter as App and Context carry it.
//
// The prompts themselves are tested in package prompt.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agilira/orpheus"
	"github.com/agilira/orpheus/prompt"
)

// testPrompter creates a Prompter backed by the given input string.
// fd=-1 disables terminal raw mode (AskSecret reads in plain text).
func testPrompter(input string) (*prompt.Terminal, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := prompt.NewFrom(strings.NewReader(input), out, -1)
	return p, out
}

// --- Wiring tests: Context + App ---

func TestContext_PrompterNilByDefault(t *testing.T) {
	app := orpheus.New("test")
	var captured prompt.Prompter
	app.Command("cmd", "test", func(ctx *orpheus.Context) error {
		captured = ctx.Prompter()
		return nil
	})
	if err := app.Run([]string{"cmd"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured != nil {
		t.Error("prompter should be nil by default")
	}
}

func TestContext_PrompterWiredFromApp(t *testing.T) {
	p, _ := testPrompter("")
	app := orpheus.New("test").SetPrompter(p)
	var captured prompt.Prompter
	app.Command("cmd", "test", func(ctx *orpheus.Context) error {
		captured = ctx.Prompter()
		return nil
	})
	if err := app.Run([]string{"cmd"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured == nil {
		t.Fatal("prompter should not be nil when set on App")
	}
	if captured != p {
		t.Error("prompter should be the same instance set on App")
	}
}

func TestContext_RequirePrompter_WhenNil(t *testing.T) {
	app := orpheus.New("test")
	var reqErr error
	app.Command("cmd", "test", func(ctx *orpheus.Context) error {
		_, reqErr = ctx.RequirePrompter()
		return nil
	})
	if err := app.Run([]string{"cmd"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqErr == nil {
		t.Fatal("RequirePrompter should return error when no prompter is set")
	}
}

func TestContext_RequirePrompter_WhenSet(t *testing.T) {
	p, _ := testPrompter("")
	app := orpheus.New("test").SetPrompter(p)
	var got prompt.Prompter
	var reqErr error
	app.Command("cmd", "test", func(ctx *orpheus.Context) error {
		got, reqErr = ctx.RequirePrompter()
		return nil
	})
	if err := app.Run([]string{"cmd"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqErr != nil {
		t.Fatalf("unexpected error from RequirePrompter: %v", reqErr)
	}
	if got != p {
		t.Error("RequirePrompter should return the configured prompter")
	}
}
