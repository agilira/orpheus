// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"bytes"
	"testing"

	"github.com/agilira/orpheus/pkg/term"
	newterm "github.com/agilira/orpheus/term"
)

// TestWrappersCallTheirTarget checks that every function of the facade
// returns what the function it wraps returns.
func TestWrappersCallTheirTarget(t *testing.T) {
	caps := term.Caps{Color: true}
	env := func(string) (string, bool) { return "", false }
	if term.Detect(&bytes.Buffer{}, env, false) != newterm.Detect(&bytes.Buffer{}, env, false) {
		t.Error("Detect differs")
	}
	if term.FormatBytes(123456) != newterm.FormatBytes(123456) {
		t.Error("FormatBytes differs")
	}
	if term.IsSafeRune('\x1b') != newterm.IsSafeRune('\x1b') || term.IsSafeRune('a') != newterm.IsSafeRune('a') {
		t.Error("IsSafeRune differs")
	}
	if term.Paint(caps, term.Red, "x") != newterm.Paint(caps, newterm.Red, "x") {
		t.Error("Paint differs")
	}
	if term.ProgressLine("l", 1, 4) != newterm.ProgressLine("l", 1, 4) {
		t.Error("ProgressLine differs")
	}
	if term.Sanitize("a\x1b[2Jb") != newterm.Sanitize("a\x1b[2Jb") {
		t.Error("Sanitize differs")
	}
	if term.Width("héllo") != newterm.Width("héllo") {
		t.Error("Width differs")
	}
	if term.Styled(term.Bold, "s") != newterm.Styled(newterm.Bold, "s") || term.Text("s") != newterm.Text("s") {
		t.Error("Styled or Text differs")
	}
	var a, b bytes.Buffer
	t1, t2 := term.NewTable(caps, "A", "B"), newterm.NewTable(caps, "A", "B")
	t1.Row(term.Text("1"), term.Styled(term.Green, "2"))
	t2.Row(newterm.Text("1"), newterm.Styled(newterm.Green, "2"))
	if t1.Render(&a) != nil || t2.Render(&b) != nil || a.String() != b.String() {
		t.Errorf("NewTable differs: %q vs %q", a.String(), b.String())
	}
	if err := term.StartSpinner(&bytes.Buffer{}, term.Caps{}, "l", term.SpinnerOptions{}).Stop(); err != nil {
		t.Errorf("StartSpinner: %v", err)
	}
}
