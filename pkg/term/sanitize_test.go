// sanitize_test.go: tests for terminal output sanitization.
//
// WHY: table cells and status lines print names chosen by someone else
// (file names, MCP server names, labels). If those bytes reach the terminal
// unchanged, a crafted name can recolor a line, move the cursor, erase the
// real status or reorder text with bidi overrides. The cases below are the
// concrete ways a hostile name can lie on screen.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"testing"
	"unicode/utf8"

	"github.com/agilira/orpheus/pkg/term"
)

func TestSanitizeKeepsPrintableText(t *testing.T) {
	cases := []string{
		"",
		"~/.claude/skills",
		`C:\Users\antonio\.mcp.json`,
		"perché caffè",
		"模型 модель",
		"a b\u00a0c",
	}
	for _, in := range cases {
		if got := term.Sanitize(in); got != in {
			t.Errorf("Sanitize(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestSanitizeEscapesHostileSequences(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ANSI color", "evil\x1b[32m ok", `evil\x1b[32m ok`},
		{"cursor up and erase line", "x\x1b[1A\x1b[2K", `x\x1b[1A\x1b[2K`},
		{"OSC title and hyperlink", "\x1b]8;;http://e\x07a", `\x1b]8;;http://e\x07a`},
		{"C1 CSI", "a\u009b31mb", `a\u009b31mb`},
		{"newline forges a row", "a\nok  mcp", `a\x0aok  mcp`},
		{"carriage return overwrites", "DRIFT\rok   ", `DRIFT\x0dok   `},
		{"tab breaks alignment", "a\tb", `a\x09b`},
		{"backspace", "ok\b\bNO", `ok\x08\x08NO`},
		{"DEL", "a\x7fb", `a\x7fb`},
		{"NUL", "a\x00b", `a\x00b`},
		{"bidi override", "txt.\u202egpj.exe", `txt.\u202egpj.exe`},
		{"bidi isolate", "a\u2066b\u2069", `a\u2066b\u2069`},
		{"zero width space", "git\u200bhub", `git\u200bhub`},
		{"byte order mark", "\ufeffname", `\ufeffname`},
		{"line separator", "a\u2028b", `a\u2028b`},
		{"invisible tag characters", "ok\U000e0041\U000e0042", `ok\U000e0041\U000e0042`},
		{"invalid UTF-8", "a\xffb\xc3", `a\xffb\xc3`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := term.Sanitize(tc.in); got != tc.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func FuzzSanitize(f *testing.F) {
	seeds := []string{
		"", "plain", "\x1b[0m", "\x1b]0;title\x07", "\u009b2J", "\u202e",
		"\xff\xfe", "a\r\nb", "\x00\x7f\u0085", "\U0001F600", "\ufeff\u200d",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := term.Sanitize(in)
		if !utf8.ValidString(out) {
			t.Fatalf("Sanitize(%q) = %q is not valid UTF-8", in, out)
		}
		for _, r := range out {
			if !term.IsSafeRune(r) {
				t.Fatalf("Sanitize(%q) = %q contains unsafe rune %U", in, out, r)
			}
		}
		if again := term.Sanitize(out); again != out {
			t.Fatalf("Sanitize is not idempotent: %q -> %q", out, again)
		}
		if len(out) > 6*len(in) {
			t.Fatalf("Sanitize(%q) grew to %d bytes", in, len(out))
		}
	})
}

func TestIsSafeRune(t *testing.T) {
	unsafe := []rune{0x00, 0x1b, 0x7f, 0x85, 0x9b, 0x200b, 0x202e, 0x2028, 0x2029, 0xfeff}
	for _, r := range unsafe {
		if term.IsSafeRune(r) {
			t.Errorf("IsSafeRune(%U) = true, want false", r)
		}
	}
	for _, r := range "aZ09 ~\\/é模\u00a0\ufffd" {
		if !term.IsSafeRune(r) {
			t.Errorf("IsSafeRune(%U) = false, want true", r)
		}
	}
}
