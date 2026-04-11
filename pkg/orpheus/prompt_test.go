// prompt_test.go: comprehensive tests for the interactive prompt system.
//
// WHY: every public method of TerminalPrompter must be exercised with
// deterministic input. Tests use NewPrompterFrom with strings.NewReader
// and bytes.Buffer to avoid any real terminal dependency.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/orpheus"
)

// testPrompter creates a Prompter backed by the given input string.
// fd=-1 disables terminal raw mode (AskSecret reads in plain text).
func testPrompter(input string) (*orpheus.TerminalPrompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := orpheus.NewPrompterFrom(strings.NewReader(input), out, -1)
	return p, out
}

// --- Ask tests ---

func TestAsk_UserEntersValue(t *testing.T) {
	p, out := testPrompter("Antonio\n")
	got, err := p.Ask("Your name", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Antonio" {
		t.Errorf("got %q, want %q", got, "Antonio")
	}
	if !strings.Contains(out.String(), "Your name:") {
		t.Errorf("output missing prompt, got %q", out.String())
	}
}

func TestAsk_EmptyReturnsDefault(t *testing.T) {
	p, _ := testPrompter("\n")
	got, err := p.Ask("Your name", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "World" {
		t.Errorf("got %q, want %q", got, "World")
	}
}

func TestAsk_DefaultShownInPrompt(t *testing.T) {
	p, out := testPrompter("\n")
	_, err := p.Ask("Timezone", "Europe/Rome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "[Europe/Rome]") {
		t.Errorf("default not shown in prompt, got %q", out.String())
	}
}

func TestAsk_NoDefaultNoSquareBrackets(t *testing.T) {
	p, out := testPrompter("test\n")
	_, err := p.Ask("Name", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "[") {
		t.Errorf("unexpected square brackets in prompt without default, got %q", out.String())
	}
}

func TestAsk_WhitespaceOnlyReturnsDefault(t *testing.T) {
	p, _ := testPrompter("   \n")
	got, err := p.Ask("Name", "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestAsk_EOFReturnsDefault(t *testing.T) {
	// WHY: empty reader = EOF immediately. Must return default, not error.
	p, _ := testPrompter("")
	got, err := p.Ask("Name", "default-user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "default-user" {
		t.Errorf("got %q, want %q", got, "default-user")
	}
}

func TestAsk_TrimWhitespace(t *testing.T) {
	p, _ := testPrompter("  hello world  \n")
	got, err := p.Ask("Value", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// --- AskSecret tests ---

func TestAskSecret_ReadsPlainTextWhenNotTTY(t *testing.T) {
	// WHY: fd=-1 means no terminal raw mode, reads as plain text.
	p, out := testPrompter("sk-test-key-123\n")
	got, err := p.AskSecret("API key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-test-key-123" {
		t.Errorf("got %q, want %q", got, "sk-test-key-123")
	}
	if !strings.Contains(out.String(), "API key:") {
		t.Errorf("output missing prompt, got %q", out.String())
	}
}

func TestAskSecret_EmptyReturnsError(t *testing.T) {
	p, _ := testPrompter("\n")
	_, err := p.AskSecret("API key")
	if err == nil {
		t.Fatal("expected error for empty secret, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got %q", err.Error())
	}
}

func TestAskSecret_WhitespaceOnlyReturnsError(t *testing.T) {
	p, _ := testPrompter("   \n")
	_, err := p.AskSecret("Password")
	if err == nil {
		t.Fatal("expected error for whitespace-only secret, got nil")
	}
}

func TestAskSecret_EOFReturnsError(t *testing.T) {
	p, _ := testPrompter("")
	_, err := p.AskSecret("Token")
	if err == nil {
		t.Fatal("expected error on EOF, got nil")
	}
}

// --- Choose tests ---

func TestChoose_ValidSelection(t *testing.T) {
	options := []string{"OpenAI", "Anthropic", "Gemini", "Ollama"}
	p, out := testPrompter("2\n")
	got, err := p.Choose("Select provider", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// WHY: user enters 2 (1-based), we return 1 (0-based).
	if got != 1 {
		t.Errorf("got %d, want %d", got, 1)
	}
	// Verify all options displayed
	outStr := out.String()
	for i, opt := range options {
		expected := strings.TrimSpace(opt)
		if !strings.Contains(outStr, expected) {
			t.Errorf("option %d %q not in output", i, opt)
		}
	}
}

func TestChoose_FirstOption(t *testing.T) {
	p, _ := testPrompter("1\n")
	got, err := p.Choose("Pick", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestChoose_LastOption(t *testing.T) {
	p, _ := testPrompter("3\n")
	got, err := p.Choose("Pick", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestChoose_OutOfRangeTooHigh(t *testing.T) {
	p, _ := testPrompter("5\n")
	_, err := p.Choose("Pick", []string{"a", "b", "c"})
	if err == nil {
		t.Fatal("expected error for out-of-range choice")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention 'out of range', got %q", err.Error())
	}
}

func TestChoose_OutOfRangeZero(t *testing.T) {
	p, _ := testPrompter("0\n")
	_, err := p.Choose("Pick", []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for zero choice")
	}
}

func TestChoose_NegativeNumber(t *testing.T) {
	p, _ := testPrompter("-1\n")
	_, err := p.Choose("Pick", []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for negative choice")
	}
}

func TestChoose_NotANumber(t *testing.T) {
	p, _ := testPrompter("abc\n")
	_, err := p.Choose("Pick", []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for non-numeric input")
	}
	if !strings.Contains(err.Error(), "invalid choice") {
		t.Errorf("error should mention 'invalid choice', got %q", err.Error())
	}
}

func TestChoose_EmptyOptions(t *testing.T) {
	p, _ := testPrompter("1\n")
	_, err := p.Choose("Pick", []string{})
	if err == nil {
		t.Fatal("expected error for empty options")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Errorf("error should mention 'at least one', got %q", err.Error())
	}
}

func TestChoose_EOFReturnsError(t *testing.T) {
	p, _ := testPrompter("")
	_, err := p.Choose("Pick", []string{"a"})
	if err == nil {
		t.Fatal("expected error on EOF")
	}
}

// --- Confirm tests ---

func TestConfirm_ExplicitYes(t *testing.T) {
	for _, input := range []string{"y\n", "Y\n", "yes\n", "YES\n", "Yes\n"} {
		p, _ := testPrompter(input)
		got, err := p.Confirm("Continue?", false)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if !got {
			t.Errorf("input %q: got false, want true", input)
		}
	}
}

func TestConfirm_ExplicitNo(t *testing.T) {
	for _, input := range []string{"n\n", "N\n", "no\n", "NO\n", "No\n"} {
		p, _ := testPrompter(input)
		got, err := p.Confirm("Continue?", true)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if got {
			t.Errorf("input %q: got true, want false", input)
		}
	}
}

func TestConfirm_DefaultYes(t *testing.T) {
	p, out := testPrompter("\n")
	got, err := p.Confirm("Continue?", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true for default=yes with empty input")
	}
	if !strings.Contains(out.String(), "[Y/n]") {
		t.Errorf("should show [Y/n], got %q", out.String())
	}
}

func TestConfirm_DefaultNo(t *testing.T) {
	p, out := testPrompter("\n")
	got, err := p.Confirm("Continue?", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected false for default=no with empty input")
	}
	if !strings.Contains(out.String(), "[y/N]") {
		t.Errorf("should show [y/N], got %q", out.String())
	}
}

func TestConfirm_InvalidAnswer(t *testing.T) {
	p, _ := testPrompter("maybe\n")
	_, err := p.Confirm("Continue?", true)
	if err == nil {
		t.Fatal("expected error for invalid answer")
	}
	if !strings.Contains(err.Error(), "invalid answer") {
		t.Errorf("error should mention 'invalid answer', got %q", err.Error())
	}
}

func TestConfirm_EOFReturnsDefault(t *testing.T) {
	p, _ := testPrompter("")
	got, err := p.Confirm("Continue?", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected true (default) on EOF")
	}
}

// --- Wiring tests: Context + App ---

func TestContext_PrompterNilByDefault(t *testing.T) {
	app := orpheus.New("test")
	var captured orpheus.Prompter
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
	var captured orpheus.Prompter
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
	var got orpheus.Prompter
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

// --- NewTerminalPrompter constructor ---

func TestNewTerminalPrompter_NotNil(t *testing.T) {
	// WHY: smoke test that the production constructor does not panic.
	p := orpheus.NewTerminalPrompter()
	if p == nil {
		t.Fatal("NewTerminalPrompter returned nil")
	}
}

// --- Sequential prompts (multi-question interview) ---

func TestAsk_MultipleSequentialCalls(t *testing.T) {
	// WHY: simulates a real interview with multiple questions.
	input := "Antonio\nEurope/Rome\nhe\n"
	p, _ := testPrompter(input)

	name, err := p.Ask("Name", "")
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	tz, err := p.Ask("Timezone", "UTC")
	if err != nil {
		t.Fatalf("tz: %v", err)
	}
	pronoun, err := p.Ask("Pronoun", "they")
	if err != nil {
		t.Fatalf("pronoun: %v", err)
	}

	if name != "Antonio" {
		t.Errorf("name: got %q", name)
	}
	if tz != "Europe/Rome" {
		t.Errorf("tz: got %q", tz)
	}
	if pronoun != "he" {
		t.Errorf("pronoun: got %q", pronoun)
	}
}

func TestMixedPrompts_AskThenChooseThenConfirm(t *testing.T) {
	// WHY: simulates a real setup wizard with different prompt types.
	input := "Antonio\n2\ny\n"
	p, _ := testPrompter(input)

	name, err := p.Ask("Name", "")
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	choice, err := p.Choose("Provider", []string{"OpenAI", "Anthropic", "Ollama"})
	if err != nil {
		t.Fatalf("choose: %v", err)
	}
	ok, err := p.Confirm("Save?", true)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}

	if name != "Antonio" {
		t.Errorf("name: got %q", name)
	}
	if choice != 1 {
		t.Errorf("choice: got %d, want 1", choice)
	}
	if !ok {
		t.Error("confirm: got false, want true")
	}
}

// --- Interface compliance ---

func TestTerminalPrompter_ImplementsPrompter(t *testing.T) {
	// WHY: compile-time guarantee that TerminalPrompter satisfies Prompter.
	var _ orpheus.Prompter = (*orpheus.TerminalPrompter)(nil)
}
