// style.go: colored text with a fixed, small palette.
//
// WHY: colors carry meaning (ok, warning, failure, unknown) and must never be
// the only carrier of it, so the palette stays small and every span is
// closed by a reset. Text is sanitized before it is wrapped: the only escape
// sequences in the output are the ones this file writes.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

// Style selects how a span of text is rendered.
type Style uint8

// The palette. The meanings are conventions, not enforced.
const (
	None   Style = iota
	Green        // success
	Yellow       // warning
	Red          // failure
	Gray         // unknown or secondary
	Bold         // emphasis
)

const reset = "\x1b[0m"

var sgr = [...]string{
	Green:  "\x1b[32m",
	Yellow: "\x1b[33m",
	Red:    "\x1b[31m",
	Gray:   "\x1b[90m",
	Bold:   "\x1b[1m",
}

// Paint returns text sanitized and, when caps allow color, wrapped in the
// escape sequence for style. Unknown styles render as plain text.
func Paint(caps Caps, style Style, text string) string {
	safe := Sanitize(text)
	if !caps.Color || int(style) >= len(sgr) || sgr[style] == "" {
		return safe
	}
	return sgr[style] + safe + reset
}
