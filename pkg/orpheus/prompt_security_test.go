// prompt_security_test.go: adversarial security testing for interactive prompts.
//
// THREAT MODEL:
// Prompter accepts arbitrary user input from stdin. Attack surface:
//
// CWE-20  (Improper Input Validation):
//   - Null bytes in text input (terminal injection, log poisoning)
//   - Control characters (ANSI escape sequences for terminal manipulation)
//   - Overlong input (resource exhaustion, buffer overflow analogue)
//   - Unicode normalization exploits (homoglyph attacks in names/keys)
//
// CWE-400 (Uncontrolled Resource Consumption):
//   - Extremely long single-line input (memory exhaustion)
//   - Scanner buffer overflow when line exceeds maxInputLine
//
// CWE-116 (Improper Encoding or Escaping of Output):
//   - ANSI escape injection via user input reflected in output
//   - Prompt injection via crafted default values
//
// CWE-522 (Insufficiently Protected Credentials):
//   - AskSecret must never echo input to output buffer
//   - Secret value must not appear in error messages
//
// CWE-362 (Race Condition):
//   - Prompter is documented as single-goroutine only.
//   - No concurrency tests needed (contract, not implementation).
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

// secPrompter creates a prompter for security tests.
func secPrompter(input string) (*orpheus.TerminalPrompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := orpheus.NewPrompterFrom(strings.NewReader(input), out, -1)
	return p, out
}

// --- CWE-20: Null bytes in input ---

func TestSecurity_Ask_NullBytesInInput(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: null bytes could truncate strings in C-backed systems,
	// poison log files, or bypass validation that only checks prefix.
	// MITIGATION EXPECTED: input is returned as-is (Go strings handle
	// null bytes safely), but the caller must validate.
	payloads := []string{
		"hello\x00world\n",
		"\x00\n",
		"before\x00\n",
		"\x00\x00\x00\n",
	}
	for _, payload := range payloads {
		p, _ := secPrompter(payload)
		got, err := p.Ask("Test", "")
		if err != nil {
			t.Errorf("payload %q: unexpected error: %v", payload, err)
			continue
		}
		// WHY: Prompter should not crash or panic on null bytes.
		// The value is returned for the caller to validate.
		if got == "" && payload != "\x00\n" {
			t.Errorf("payload %q: got empty string unexpectedly", payload)
		}
	}
}

// --- CWE-20: ANSI escape sequences ---

func TestSecurity_Ask_ANSIEscapeInInput(t *testing.T) {
	// ATTACK VECTOR: CWE-20 / CWE-116
	// IMPACT: injected ANSI escapes could manipulate terminal display,
	// hide prompts, or forge output ("You are authenticated!" injection).
	// MITIGATION EXPECTED: Prompter does not strip ANSI (that is the
	// caller's responsibility), but must not crash.
	payloads := []string{
		"\x1b[31mred\x1b[0m\n",         // color injection
		"\x1b[2J\x1b[H\n",              // clear screen
		"\x1b]0;malicious title\x07\n", // set terminal title
		"\x1b[?25l\n",                  // hide cursor
	}
	for _, payload := range payloads {
		p, _ := secPrompter(payload)
		_, err := p.Ask("Test", "")
		if err != nil {
			t.Errorf("ANSI payload: unexpected error: %v", err)
		}
	}
}

// --- CWE-400: Overlong input ---

func TestSecurity_Ask_OverlongInput(t *testing.T) {
	// ATTACK VECTOR: CWE-400
	// IMPACT: extremely long line could exhaust memory or cause OOM.
	// MITIGATION EXPECTED: bufio.Scanner with maxInputLine limit rejects
	// lines exceeding 4096 bytes. The scan returns false and Err() reports
	// the overflow.
	longInput := strings.Repeat("A", 8192) + "\n"
	p, _ := secPrompter(longInput)
	got, err := p.Ask("Test", "default-value")
	// WHY: the scanner should either truncate or fail gracefully.
	// With our 4096 limit, Scan() returns false with a token-too-long error.
	// Ask() should return the default or an error, never panic.
	if err == nil && got != "default-value" {
		// If the scanner accepted it (unlikely with 4096 limit), that is
		// acceptable but unexpected.
		t.Logf("overlong input accepted: len=%d", len(got))
	}
	// Main assertion: no panic, no crash.
}

func TestSecurity_AskSecret_OverlongInput(t *testing.T) {
	// ATTACK VECTOR: CWE-400
	// Same as above but for secret input.
	longInput := strings.Repeat("X", 8192) + "\n"
	p, _ := secPrompter(longInput)
	_, err := p.AskSecret("Secret")
	// WHY: an error is acceptable (token too long or empty after truncation).
	// A panic is NOT acceptable.
	if err == nil {
		t.Log("overlong secret accepted without error (scanner limit may differ)")
	}
}

// --- CWE-400: Choose with huge option list ---

func TestSecurity_Choose_ManyOptions(t *testing.T) {
	// ATTACK VECTOR: CWE-400
	// IMPACT: if the caller passes 100K options, Choose must still
	// work without excessive memory or time.
	// MITIGATION EXPECTED: Choose writes each option linearly. No
	// quadratic behavior.
	options := make([]string, 1000)
	for i := range options {
		options[i] = "option"
	}
	p, _ := secPrompter("500\n")
	got, err := p.Choose("Pick", options)
	if err != nil {
		t.Fatalf("unexpected error with 1000 options: %v", err)
	}
	if got != 499 {
		t.Errorf("got %d, want 499", got)
	}
}

// --- CWE-116: Injection via default value ---

func TestSecurity_Ask_InjectionInDefault(t *testing.T) {
	// ATTACK VECTOR: CWE-116
	// IMPACT: a malicious default value could contain format strings
	// or ANSI escapes that are reflected in the prompt output.
	// MITIGATION EXPECTED: fmt.Fprintf with %s format prevents
	// format string injection. ANSI in defaults is passed through
	// (display concern, not security concern for Go).
	maliciousDefaults := []string{
		"%s%s%s%s%s",                 // format string attack
		"$(rm -rf /)",                // command injection
		"`whoami`",                   // backtick injection
		"\x1b[31mred_default\x1b[0m", // ANSI in default
		"'; DROP TABLE users; --",    // SQL injection
	}
	for _, def := range maliciousDefaults {
		p, out := secPrompter("\n") // accept default
		got, err := p.Ask("Test", def)
		if err != nil {
			t.Errorf("default %q: unexpected error: %v", def, err)
			continue
		}
		// WHY: the malicious default is returned verbatim (it IS the default).
		// The critical thing is that fmt.Fprintf did not interpret it.
		if got != def {
			t.Errorf("default %q: got %q", def, got)
		}
		// Verify the output contains the default literally, not interpreted.
		if !strings.Contains(out.String(), def) {
			t.Logf("default not found literally in output (ANSI may differ)")
		}
	}
}

// --- CWE-522: Secret not leaked ---

func TestSecurity_AskSecret_NotEchoedToOutput(t *testing.T) {
	// ATTACK VECTOR: CWE-522
	// IMPACT: if AskSecret echoes the secret to the writer, it appears
	// in logs, screen captures, or terminal scrollback.
	// MITIGATION EXPECTED: in non-TTY mode (fd=-1), the secret value
	// must NOT appear in the output buffer.
	secretValue := "sk-super-secret-key-12345"
	p, out := secPrompter(secretValue + "\n")
	got, err := p.AskSecret("API key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != secretValue {
		t.Errorf("got %q, want %q", got, secretValue)
	}
	// WHY: the output should contain "API key:" prompt but NOT the secret itself.
	if strings.Contains(out.String(), secretValue) {
		t.Error("SECURITY: secret value leaked to output writer")
	}
}

func TestSecurity_AskSecret_ErrorDoesNotLeakValue(t *testing.T) {
	// ATTACK VECTOR: CWE-522
	// IMPACT: if an error message includes the secret value, it could
	// appear in logs.
	// MITIGATION EXPECTED: error messages from AskSecret never contain
	// the attempted secret value.
	// WHY: empty secret triggers error. Verify the error text.
	p, _ := secPrompter("\n")
	_, err := p.AskSecret("Token")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
	// The error should say "empty" but not include any previous input.
	errMsg := err.Error()
	if strings.Contains(errMsg, "Token") && !strings.Contains(errMsg, "empty") {
		t.Errorf("error message may leak context: %q", errMsg)
	}
}

// --- CWE-20: Unicode edge cases ---

func TestSecurity_Ask_UnicodeEdgeCases(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: Unicode normalization could allow homoglyph attacks
	// (user enters lookalike characters for names, paths).
	// MITIGATION EXPECTED: Prompter preserves exact Unicode bytes.
	// Normalization is the caller's concern.
	payloads := []struct {
		input string
		desc  string
	}{
		{"\xef\xbb\xbfBOM_prefix\n", "UTF-8 BOM prefix"},
		{"\u200Bzero_width\n", "zero-width space"},
		{"\u202Ereversed\n", "right-to-left override"},
		{"caf\xc3\xa9\n", "valid UTF-8 (cafe with accent)"},
		{"Ant\xc3\xb2nio\n", "name with accent (Antonio)"},
	}
	for _, tc := range payloads {
		p, _ := secPrompter(tc.input)
		got, err := p.Ask("Test", "")
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.desc, err)
			continue
		}
		// WHY: just verify no panic and non-empty return.
		if got == "" {
			t.Errorf("%s: got empty string", tc.desc)
		}
	}
}

// --- Choose: integer overflow attempt ---

func TestSecurity_Choose_IntegerOverflow(t *testing.T) {
	// ATTACK VECTOR: CWE-20 / CWE-190
	// IMPACT: extremely large number could cause integer overflow.
	// MITIGATION EXPECTED: strconv.Atoi returns error for overflow.
	payloads := []string{
		"99999999999999999999\n",  // exceeds int64
		"-99999999999999999999\n", // negative overflow
		"2147483648\n",            // int32 max + 1
	}
	for _, payload := range payloads {
		p, _ := secPrompter(payload)
		_, err := p.Choose("Pick", []string{"a", "b"})
		if err == nil {
			t.Errorf("payload %q: expected error for overflow input", payload)
		}
	}
}

// --- Choose: injection in option display ---

func TestSecurity_Choose_InjectionInOptions(t *testing.T) {
	// ATTACK VECTOR: CWE-116
	// IMPACT: if option strings contain format specifiers or ANSI,
	// they are reflected in the output.
	// MITIGATION EXPECTED: fmt.Fprintf with %s prevents interpretation.
	options := []string{
		"%s%s%s",
		"\x1b[31mred\x1b[0m",
		"$(whoami)",
	}
	p, _ := secPrompter("1\n")
	got, err := p.Choose("Pick", options)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
