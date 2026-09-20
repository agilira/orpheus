// prompt_limit_test.go: the input line limit must hold on the production path.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// askWithStdin runs Ask against a real NewTerminalPrompter whose stdin is the
// given input, and reports what it returned.
func askWithStdin(t *testing.T, input string) (string, error) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})

	go func() {
		_, _ = io.WriteString(w, input)
		_ = w.Close()
	}()

	p := NewTerminalPrompter()
	p.writer = io.Discard
	return p.Ask("Name", "")
}

// A line beyond the documented limit must be refused, not buffered.
func TestTerminalPrompterEnforcesLineLimit(t *testing.T) {
	long := strings.Repeat("A", maxInputLine+1)

	got, err := askWithStdin(t, long+"\n")
	if err == nil {
		t.Fatalf("Ask accepted a %d-byte line; the limit is %d and applies only to NewPrompterFrom (got %d bytes back)",
			len(long), maxInputLine, len(got))
	}
}

// A line within the limit still works.
func TestTerminalPrompterAcceptsOrdinaryInput(t *testing.T) {
	got, err := askWithStdin(t, "antonio\n")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if got != "antonio" {
		t.Fatalf("Ask = %q, want %q", got, "antonio")
	}
}

// The injectable constructor enforces the same limit, as it already did.
func TestPrompterFromEnforcesLineLimit(t *testing.T) {
	long := strings.Repeat("A", maxInputLine+1)
	p := NewPrompterFrom(strings.NewReader(long+"\n"), &bytes.Buffer{}, -1)

	if _, err := p.Ask("Name", ""); err == nil {
		t.Fatal("Ask accepted a line beyond the limit")
	}
}
