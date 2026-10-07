// table_test.go: tests for aligned tables.
//
// WHY: a status table is where a user decides whether something is safe.
// A cell must never be able to add a row, shift a column or recolor its
// neighbours, whatever bytes it contains.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/agilira/orpheus/pkg/term"
)

var sgrPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func render(t *testing.T, tb term.Table) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tb.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

func TestTableAlignsColumns(t *testing.T) {
	tb := term.NewTable(term.Caps{}, "STATE", "KIND", "PROBE")
	tb.Row(term.Text("ok"), term.Text("mcp"), term.Text("~/.claude.json"))
	tb.Row(term.Text("MISSING"), term.Text("file"), term.Text("~/.gemini/GEMINI.md"))
	want := "STATE    KIND  PROBE\n" +
		"ok       mcp   ~/.claude.json\n" +
		"MISSING  file  ~/.gemini/GEMINI.md\n"
	if got := render(t, tb); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTableWithoutHeaders(t *testing.T) {
	tb := term.NewTable(term.Caps{})
	tb.Row(term.Text("a"), term.Text("b"))
	if got := render(t, tb); got != "a  b\n" {
		t.Fatalf("got %q", got)
	}
}

func TestTableEmptyWritesNothing(t *testing.T) {
	if got := render(t, term.NewTable(term.Caps{})); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestTableRaggedRowsKeepEveryCell(t *testing.T) {
	tb := term.NewTable(term.Caps{}, "A")
	tb.Row(term.Text("1"), term.Text("extra"))
	tb.Row()
	want := "A\n1  extra\n\n"
	if got := render(t, tb); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTableCellCannotForgeRows(t *testing.T) {
	tb := term.NewTable(term.Caps{}, "STATE", "PROBE")
	tb.Row(term.Text("DRIFT"), term.Text("x\nok     ~/.mcp.json\r\x1b[2K"))
	got := render(t, tb)
	if lines := strings.Count(got, "\n"); lines != 2 {
		t.Fatalf("a cell added lines: %q", got)
	}
	if strings.ContainsAny(got, "\r\x1b") {
		t.Fatalf("raw control reached the output: %q", got)
	}
}

func TestTableColorDoesNotShiftColumns(t *testing.T) {
	build := func(c term.Caps) term.Table {
		tb := term.NewTable(c, "STATE", "PROBE")
		tb.Row(term.Styled(term.Red, "DRIFT"), term.Text("a"))
		tb.Row(term.Styled(term.Green, "ok"), term.Text("b"))
		return tb
	}
	plain := render(t, build(term.Caps{}))
	colored := render(t, build(colorCaps))
	if colored == plain {
		t.Fatal("colored table has no color")
	}
	if stripped := sgrPattern.ReplaceAllString(colored, ""); stripped != plain {
		t.Fatalf("color changed the layout:\n%q\n%q", stripped, plain)
	}
}

func TestTableHeadersAreBoldWithColor(t *testing.T) {
	tb := term.NewTable(colorCaps, "A")
	if got := render(t, tb); got != "\x1b[1mA\x1b[0m\n" {
		t.Fatalf("got %q", got)
	}
}

func TestTableWideAndCombiningRunes(t *testing.T) {
	tb := term.NewTable(term.Caps{}, "NAME", "X")
	tb.Row(term.Text("模型"), term.Text("1"))
	tb.Row(term.Text("cafe\u0301"), term.Text("2"))
	want := "NAME  X\n模型  1\ncafe\u0301  2\n"
	if got := render(t, tb); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestTableRenderReturnsWriteError(t *testing.T) {
	tb := term.NewTable(term.Caps{}, "A")
	if err := tb.Render(failWriter{}); err == nil {
		t.Fatal("Render swallowed the write error")
	}
}

func TestWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"模型", 4},
		{"한국", 4},
		{"ｆｕｌｌ", 8},
		{"e\u0301", 1},
		{"\U0001F600", 2},
	}
	for _, tc := range cases {
		if got := term.Width(tc.in); got != tc.want {
			t.Errorf("Width(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func FuzzTable(f *testing.F) {
	f.Add("ok", "\x1b[31mDRIFT\n", "模型\u202e")
	f.Add("", "\r", "\xff")
	f.Fuzz(func(t *testing.T, a, b, c string) {
		build := func(caps term.Caps) string {
			tb := term.NewTable(caps, "H1", "H2")
			tb.Row(term.Styled(term.Red, a), term.Text(b))
			tb.Row(term.Text(c))
			var buf bytes.Buffer
			if err := tb.Render(&buf); err != nil {
				t.Fatalf("Render: %v", err)
			}
			return buf.String()
		}
		plain := build(term.Caps{})
		if strings.Count(plain, "\n") != 3 {
			t.Fatalf("cells changed the row count: %q", plain)
		}
		if strings.ContainsAny(plain, "\x1b\r") {
			t.Fatalf("raw control in plain output: %q", plain)
		}
		colored := build(colorCaps)
		if sgrPattern.ReplaceAllString(colored, "") != plain {
			t.Fatalf("color changed the layout: %q vs %q", colored, plain)
		}
	})
}
