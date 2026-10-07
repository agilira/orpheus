// style_test.go: tests for colored text.
//
// WHY: a color is only allowed to decorate text that has already been made
// safe, and only when the writer can display it. These tests pin both
// rules, so that a hostile name cannot smuggle its own escapes inside a
// legitimate colored span.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/term"
)

var colorCaps = term.Caps{Color: true, Animate: true}

func TestPaintWithColor(t *testing.T) {
	cases := []struct {
		style term.Style
		want  string
	}{
		{term.Green, "\x1b[32mok\x1b[0m"},
		{term.Yellow, "\x1b[33mok\x1b[0m"},
		{term.Red, "\x1b[31mok\x1b[0m"},
		{term.Gray, "\x1b[90mok\x1b[0m"},
		{term.Bold, "\x1b[1mok\x1b[0m"},
		{term.None, "ok"},
	}
	for _, tc := range cases {
		if got := term.Paint(colorCaps, tc.style, "ok"); got != tc.want {
			t.Errorf("Paint(%v) = %q, want %q", tc.style, got, tc.want)
		}
	}
}

func TestPaintWithoutColorIsPlain(t *testing.T) {
	if got := term.Paint(term.Caps{}, term.Red, "DRIFT"); got != "DRIFT" {
		t.Fatalf("Paint without color = %q, want plain text", got)
	}
}

func TestPaintUnknownStyleIsPlain(t *testing.T) {
	if got := term.Paint(colorCaps, term.Style(200), "x"); got != "x" {
		t.Fatalf("Paint with unknown style = %q, want plain text", got)
	}
}

func TestPaintSanitizesBeforeColoring(t *testing.T) {
	got := term.Paint(colorCaps, term.Green, "evil\x1b[0m\x1b[31mDRIFT")
	want := "\x1b[32m" + `evil\x1b[0m\x1b[31mDRIFT` + "\x1b[0m"
	if got != want {
		t.Fatalf("Paint = %q, want %q", got, want)
	}
	if strings.Count(got, "\x1b") != 2 {
		t.Fatalf("Paint let a raw escape through: %q", got)
	}
}

func FuzzPaint(f *testing.F) {
	f.Add("ok", uint8(term.Green))
	f.Add("\x1b[2J\u009b0m", uint8(term.Red))
	f.Fuzz(func(t *testing.T, in string, style uint8) {
		plain := term.Paint(term.Caps{}, term.Style(style), in)
		if plain != term.Sanitize(in) {
			t.Fatalf("plain Paint(%q) = %q, want Sanitize", in, plain)
		}
		colored := term.Paint(colorCaps, term.Style(style), in)
		if n := strings.Count(colored, "\x1b"); n != 0 && n != 2 {
			t.Fatalf("Paint(%q) = %q has %d escapes, want 0 or 2", in, colored, n)
		}
	})
}
