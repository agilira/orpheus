// expose_test.go: tests for the Prometheus text exposition primitives.
//
// WHY: label values and help text are the only caller-supplied strings that
// reach the endpoint. A value such as `x"} 1\nadmin_ok 1` must stay inside
// its quotes: if it escapes them, a scraper records series the program
// never produced. The fuzz test parses every rendered line back and demands
// the exact original value.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"math"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEscapeLabelValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{`a"b`, `a\"b`},
		{`a\b`, `a\\b`},
		{"a\nb", `a\nb`},
		{"x\"} 1\nadmin_ok 1", `x\"} 1\nadmin_ok 1`},
		{"a\xffb", "a\ufffdb"},
		{"tab\tcr\r", "tab\tcr\r"},
	}
	for _, tc := range cases {
		if got := escapeLabelValue(tc.in); got != tc.want {
			t.Errorf("escapeLabelValue(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEscapeHelp(t *testing.T) {
	if got := escapeHelp("a\\b\nc\"d\xff"); got != `a\\b\nc"d`+"\ufffd" {
		t.Fatalf("escapeHelp = %q", got)
	}
}

func TestFormatValue(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-2.5, "-2.5"},
		{1e21, "1e+21"},
		{math.Inf(1), "+Inf"},
		{math.Inf(-1), "-Inf"},
		{math.NaN(), "NaN"},
	}
	for _, tc := range cases {
		if got := formatValue(tc.in); got != tc.want {
			t.Errorf("formatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWriteSample(t *testing.T) {
	var b strings.Builder
	writeSample(&b, "m", []string{"a", "b"}, []string{"1", `"`}, "", "", 3)
	writeSample(&b, "m", nil, nil, "", "", 1)
	writeSample(&b, "h_bucket", []string{"a"}, []string{"x"}, "le", "+Inf", 2)
	want := "m{a=\"1\",b=\"\\\"\"} 3\nm 1\nh_bucket{a=\"x\",le=\"+Inf\"} 2\n"
	if b.String() != want {
		t.Fatalf("got %q, want %q", b.String(), want)
	}
}

func TestWriteHeader(t *testing.T) {
	var b strings.Builder
	writeHeader(&b, "m", "line one\nline two", "counter")
	want := "# HELP m line one\\nline two\n# TYPE m counter\n"
	if b.String() != want {
		t.Fatalf("got %q, want %q", b.String(), want)
	}
}

// samplePattern accepts one sample line with a single label, as written by
// writeSample: the value group allows only escapes and non-special bytes.
var samplePattern = regexp.MustCompile(`^m\{l="((?:[^"\\\n]|\\[\\"n])*)"\} 1$`)

func unescapeLabelValue(s string) string {
	r := strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\n`, "\n")
	return r.Replace(s)
}

func FuzzLabelValueStaysQuoted(f *testing.F) {
	for _, s := range []string{"", "x\"} 1\nadmin 1", `\`, `\"`, "\\\n", "\xff\"", "a\r\n# TYPE x gauge"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, v string) {
		var b strings.Builder
		writeSample(&b, "m", []string{"l"}, []string{v}, "", "", 1)
		out := strings.TrimSuffix(b.String(), "\n")
		if strings.Contains(out, "\n") {
			t.Fatalf("value %q produced more than one line: %q", v, out)
		}
		m := samplePattern.FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("value %q produced a malformed line: %q", v, out)
		}
		if got, want := unescapeLabelValue(m[1]), strings.ToValidUTF8(v, "\ufffd"); got != want {
			t.Fatalf("round trip of %q gave %q, want %q", v, got, want)
		}
		if !utf8.ValidString(out) {
			t.Fatalf("line is not valid UTF-8: %q", out)
		}
	})
}

func FuzzHelpStaysOnOneLine(f *testing.F) {
	f.Add("x\n# TYPE evil counter\nevil 1")
	f.Fuzz(func(t *testing.T, help string) {
		var b strings.Builder
		writeHeader(&b, "m", help, "gauge")
		if lines := strings.Count(b.String(), "\n"); lines != 2 {
			t.Fatalf("help %q produced %d lines", help, lines)
		}
		if !utf8.ValidString(b.String()) {
			t.Fatalf("header is not valid UTF-8: %q", b.String())
		}
	})
}
