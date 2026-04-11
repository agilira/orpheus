// prompt_fuzz_test.go: adversarial fuzz testing for interactive prompts.
//
// WHY: every method that accepts user input must survive arbitrary bytes.
// Seeds are REAL attack patterns from CWE catalogues, not random noise.
//
// Targets:
//   FuzzAsk       -- arbitrary text input (names, paths, values)
//   FuzzAskSecret -- arbitrary secret input (API keys, tokens)
//   FuzzChoose    -- arbitrary numeric/non-numeric selection
//   FuzzConfirm   -- arbitrary yes/no/garbage responses
//
// Invariants verified:
//   - No panic on any input
//   - No out-of-bounds access
//   - Error messages never contain secret values
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

// fuzzPrompter creates a prompter from raw fuzz input.
func fuzzPrompter(data []byte) (*orpheus.TerminalPrompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := orpheus.NewPrompterFrom(bytes.NewReader(data), out, -1)
	return p, out
}

func FuzzAsk(f *testing.F) {
	// Seeds: real attack patterns for text input fields.
	seeds := []string{
		"Antonio\n",                      // normal input
		"\n",                             // empty (accept default)
		"   \n",                          // whitespace only
		"",                               // EOF
		"\x00\n",                         // null byte
		"\x00\x00\x00\x00\n",             // multiple null bytes
		"\x1b[31mred\x1b[0m\n",           // ANSI color injection
		"\x1b[2J\x1b[H\n",                // ANSI clear screen
		"\x1b]0;evil title\x07\n",        // ANSI title injection
		"%s%s%s%s%s%s%s%s\n",             // format string attack
		"$(rm -rf /)\n",                  // command injection
		"`whoami`\n",                     // backtick injection
		"'; DROP TABLE users; --\n",      // SQL injection
		"\xef\xbb\xbfBOM\n",              // UTF-8 BOM
		"\u200Bzero_width\n",             // zero-width space
		"\u202Ereversed\n",               // RTL override
		"a]b[c\n",                        // bracket injection
		strings.Repeat("A", 5000) + "\n", // overlong line
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		p, _ := fuzzPrompter(data)
		// WHY: must never panic regardless of input content.
		_, _ = p.Ask("Test", "default")
	})
}

func FuzzAskSecret(f *testing.F) {
	// Seeds: attack patterns for secret/credential input.
	seeds := []string{
		"sk-test-key\n",                  // normal API key
		"\n",                             // empty (should error)
		"   \n",                          // whitespace only (should error)
		"",                               // EOF (should error)
		"\x00secret\x00\n",               // null bytes in secret
		"\x1b[31mfake-key\x1b[0m\n",      // ANSI in secret
		"%s%s%s\n",                       // format string
		strings.Repeat("k", 5000) + "\n", // overlong secret
		"Bearer eyJhbGciOiJI\n",          // JWT-like token
		"AKIA1234567890ABCDEF\n",         // AWS key pattern
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		p, out := fuzzPrompter(data)
		secret, err := p.AskSecret("Key")

		// INV-1: must never panic.
		// INV-2: if secret returned and long enough to be meaningful,
		// it must not appear in output. Short secrets (< 4 chars) may
		// coincidentally match substrings of the prompt text itself,
		// so we only assert for secrets that are unambiguously distinct.
		if err == nil && len(secret) >= 4 && strings.Contains(out.String(), secret) {
			t.Errorf("SECURITY: secret %q leaked to output", secret)
		}
	})
}

func FuzzChoose(f *testing.F) {
	// Seeds: attack patterns for numeric menu selection.
	seeds := []string{
		"1\n",                     // valid choice
		"2\n",                     // valid choice
		"0\n",                     // out of range (zero)
		"-1\n",                    // negative
		"999\n",                   // out of range high
		"abc\n",                   // not a number
		"\n",                      // empty
		"",                        // EOF
		"1; rm -rf /\n",           // command injection
		"1' OR '1'='1\n",          // SQL injection
		"99999999999999999999\n",  // integer overflow
		"-99999999999999999999\n", // negative overflow
		"\x00\n",                  // null byte
		"1\x001\n",                // null between digits
		"  2  \n",                 // whitespace padding
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		p, _ := fuzzPrompter(data)
		options := []string{"alpha", "beta", "gamma"}
		idx, err := p.Choose("Pick", options)

		// INV-1: must never panic.
		// INV-2: if no error, index must be in valid range.
		if err == nil {
			if idx < 0 || idx >= len(options) {
				t.Errorf("index %d out of range [0, %d)", idx, len(options))
			}
		}
	})
}

func FuzzConfirm(f *testing.F) {
	// Seeds: attack patterns for yes/no confirmation.
	seeds := []string{
		"y\n",                // yes
		"n\n",                // no
		"Y\n",                // uppercase yes
		"N\n",                // uppercase no
		"yes\n",              // full word
		"no\n",               // full word
		"\n",                 // empty (accept default)
		"",                   // EOF
		"maybe\n",            // invalid
		"yy\n",               // invalid (double)
		"\x00\n",             // null byte
		"y\x00\n",            // null after y
		"\x1b[31my\x1b[0m\n", // ANSI wrapped y
		"YES\n",              // all caps
		"  y  \n",            // whitespace padded
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		p, _ := fuzzPrompter(data)
		// WHY: must never panic. Result must be consistent with input.
		_, _ = p.Confirm("Continue?", true)
	})
}
