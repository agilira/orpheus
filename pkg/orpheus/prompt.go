// prompt.go: interactive terminal prompts for CLI applications.
//
// WHY: CLI applications often need to ask users for input during setup,
// onboarding, or interactive workflows. This module provides a testable,
// injectable interface for terminal interaction. Production code uses
// TerminalPrompter (real stdin/stdout); tests inject a mock.
//
// DESIGN: Prompter is an interface so handlers never depend on a concrete
// terminal. The same handler can run in a unit test with predetermined
// answers, in a CI pipeline with scripted input, or in a real terminal
// with a human. This follows Orpheus's interface-first doctrine.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Prompter defines the interface for interactive user prompts.
// Implementations must be safe for sequential use within a single
// command handler. Concurrent use is NOT required.
type Prompter interface {
	// Ask displays a prompt and reads a single line of text.
	// If the user enters nothing, defaultVal is returned.
	// Returns an error only on I/O failure, never on empty input.
	Ask(prompt, defaultVal string) (string, error)

	// AskSecret displays a prompt and reads input without echoing.
	// WHY: API keys, passwords, and tokens must never be visible on screen.
	// Returns an error on I/O failure or if the input is empty.
	AskSecret(prompt string) (string, error)

	// Choose displays a numbered menu and returns the selected index (0-based).
	// options must contain at least one element.
	// Returns an error if the selection is out of range or on I/O failure.
	Choose(prompt string, options []string) (int, error)

	// Confirm displays a yes/no question and returns the boolean answer.
	// defaultYes controls the default when the user presses Enter:
	// true -> "[Y/n]", false -> "[y/N]".
	Confirm(prompt string, defaultYes bool) (bool, error)
}

// TerminalPrompter implements Prompter using real terminal I/O.
// It reads from an io.Reader and writes to an io.Writer, making it
// testable even though it targets real terminals in production.
//
// WHY io.Reader/io.Writer instead of *os.File: allows tests to inject
// bytes.Buffer or strings.Reader without touching any real terminal.
// The only exception is AskSecret, which needs a file descriptor for
// terminal raw mode. When fd < 0, AskSecret falls back to plain text
// reading (useful in tests and non-TTY environments like CI pipes).
type TerminalPrompter struct {
	reader  *bufio.Scanner
	writer  io.Writer
	fd      int // file descriptor for AskSecret (-1 = fallback to plain read)
	maxLine int // maximum bytes per input line (defense against CWE-400)
}

// maxInputLine is the default maximum input line length in bytes.
// WHY 4096: large enough for any reasonable user input (names, API keys,
// URLs), small enough to prevent memory exhaustion from malicious input.
const maxInputLine = 4096

// NewTerminalPrompter creates a Prompter backed by real stdin/stdout.
// This is the production constructor. Tests should use NewPrompterFrom.
func NewTerminalPrompter() *TerminalPrompter {
	return &TerminalPrompter{
		reader:  bufio.NewScanner(os.Stdin),
		writer:  os.Stdout,
		fd:      int(os.Stdin.Fd()),
		maxLine: maxInputLine,
	}
}

// NewPrompterFrom creates a Prompter backed by the given reader and writer.
// fd is the file descriptor for AskSecret; pass -1 to disable terminal
// raw mode (AskSecret will read in plain text, suitable for tests).
//
// WHY this constructor exists: unit tests inject a buffer here, so every
// method can be exercised without a real terminal.
func NewPrompterFrom(r io.Reader, w io.Writer, fd int) *TerminalPrompter {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, maxInputLine), maxInputLine)
	return &TerminalPrompter{
		reader:  scanner,
		writer:  w,
		fd:      fd,
		maxLine: maxInputLine,
	}
}

// Ask displays a prompt and reads a single line of text.
func (p *TerminalPrompter) Ask(prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		if _, err := fmt.Fprintf(p.writer, "%s [%s]: ", prompt, defaultVal); err != nil {
			return "", fmt.Errorf("prompt write failed: %w", err)
		}
	} else {
		if _, err := fmt.Fprintf(p.writer, "%s: ", prompt); err != nil {
			return "", fmt.Errorf("prompt write failed: %w", err)
		}
	}

	if !p.reader.Scan() {
		if err := p.reader.Err(); err != nil {
			return "", fmt.Errorf("read failed: %w", err)
		}
		// WHY: EOF without error means stdin was closed (pipe ended).
		// Return the default rather than failing -- allows scripted usage
		// where the pipe provides fewer lines than prompts.
		return defaultVal, nil
	}

	line := strings.TrimSpace(p.reader.Text())
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// AskSecret displays a prompt and reads input without echoing.
func (p *TerminalPrompter) AskSecret(prompt string) (string, error) {
	if _, err := fmt.Fprintf(p.writer, "%s: ", prompt); err != nil {
		return "", fmt.Errorf("prompt write failed: %w", err)
	}

	var secret string

	if p.fd >= 0 && term.IsTerminal(p.fd) {
		// WHY: term.ReadPassword puts the terminal in raw mode so keystrokes
		// are not echoed. This is the only safe way to read API keys.
		raw, err := term.ReadPassword(p.fd)
		if err != nil {
			return "", fmt.Errorf("secret read failed: %w", err)
		}
		// WHY: print newline after secret input because ReadPassword swallows it.
		if _, err := fmt.Fprintln(p.writer); err != nil {
			return "", fmt.Errorf("write newline failed: %w", err)
		}
		secret = string(raw)
	} else {
		// WHY: fallback for non-TTY environments (tests, CI, pipes).
		// Input will be visible, but the alternative is failing entirely.
		if !p.reader.Scan() {
			if err := p.reader.Err(); err != nil {
				return "", fmt.Errorf("read failed: %w", err)
			}
			return "", fmt.Errorf("secret read failed: unexpected end of input")
		}
		secret = p.reader.Text()
	}

	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("secret cannot be empty")
	}

	return secret, nil
}

// writeMenu renders a numbered option list with a trailing choice prompt.
// WHY: extracted to keep Choose below cyclomatic complexity 10.
func (p *TerminalPrompter) writeMenu(prompt string, options []string) error {
	if _, err := fmt.Fprintln(p.writer, prompt); err != nil {
		return fmt.Errorf("prompt write failed: %w", err)
	}
	for i, opt := range options {
		if _, err := fmt.Fprintf(p.writer, "  %d. %s\n", i+1, opt); err != nil {
			return fmt.Errorf("option write failed: %w", err)
		}
	}
	if _, err := fmt.Fprintf(p.writer, "Choice [1-%d]: ", len(options)); err != nil {
		return fmt.Errorf("prompt write failed: %w", err)
	}
	return nil
}

// Choose displays a numbered menu and returns the selected index (0-based).
func (p *TerminalPrompter) Choose(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("choose requires at least one option")
	}

	if err := p.writeMenu(prompt, options); err != nil {
		return 0, err
	}

	if !p.reader.Scan() {
		if err := p.reader.Err(); err != nil {
			return 0, fmt.Errorf("read failed: %w", err)
		}
		return 0, fmt.Errorf("read failed: unexpected end of input")
	}

	text := strings.TrimSpace(p.reader.Text())
	num, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("invalid choice %q: must be a number", text)
	}

	if num < 1 || num > len(options) {
		return 0, fmt.Errorf("choice %d out of range [1-%d]", num, len(options))
	}

	// WHY: return 0-based index because Go slices are 0-based.
	// The display is 1-based for human readability.
	return num - 1, nil
}

// Confirm displays a yes/no question and returns the boolean answer.
func (p *TerminalPrompter) Confirm(prompt string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	if _, err := fmt.Fprintf(p.writer, "%s %s: ", prompt, hint); err != nil {
		return false, fmt.Errorf("prompt write failed: %w", err)
	}

	if !p.reader.Scan() {
		if err := p.reader.Err(); err != nil {
			return false, fmt.Errorf("read failed: %w", err)
		}
		// WHY: EOF = accept default (same rationale as Ask).
		return defaultYes, nil
	}

	answer := strings.TrimSpace(strings.ToLower(p.reader.Text()))

	switch answer {
	case "":
		return defaultYes, nil
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid answer %q: expected y/n", answer)
	}
}
