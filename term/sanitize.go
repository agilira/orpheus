// sanitize.go: make untrusted text safe to print on a terminal.
//
// WHY: a CLI that prints names it did not choose (paths, server names,
// labels) hands those bytes to the terminal emulator, which interprets
// escape sequences, carriage returns and bidi controls. A crafted name can
// then recolor or overwrite the line that reports it. Escaping instead of
// dropping keeps the evidence visible: the reader sees that the name is odd.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsSafeRune reports whether r can reach a terminal without changing how
// the surrounding text is displayed. Control characters (C0, DEL, C1),
// format characters (bidi overrides, zero-width joiners, BOM, tag
// characters) and the Unicode line and paragraph separators are unsafe.
func IsSafeRune(r rune) bool {
	if !utf8.ValidRune(r) {
		return false
	}
	return !unicode.IsControl(r) &&
		!unicode.Is(unicode.Cf, r) &&
		!unicode.Is(unicode.Zl, r) &&
		!unicode.Is(unicode.Zp, r)
}

// Sanitize returns s with every unsafe rune and every invalid UTF-8 byte
// replaced by a visible Go-style escape (\x1b, \u202e, \U000e0041).
// The result contains only safe runes, so Sanitize is idempotent.
func Sanitize(s string) string {
	if isClean(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			writeHex(&b, `\x`, rune(s[i]), 2)
		case IsSafeRune(r):
			b.WriteRune(r)
		default:
			writeEscape(&b, r)
		}
		i += size
	}
	return b.String()
}

// isClean lets the common case return the input without allocating.
func isClean(s string) bool {
	for i, r := range s {
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(s[i:]); size == 1 {
				return false
			}
		}
		if !IsSafeRune(r) {
			return false
		}
	}
	return true
}

func writeEscape(b *strings.Builder, r rune) {
	switch {
	case r < utf8.RuneSelf:
		writeHex(b, `\x`, r, 2)
	case r <= 0xffff:
		writeHex(b, `\u`, r, 4)
	default:
		writeHex(b, `\U`, r, 8)
	}
}

// writeHex avoids fmt so that no write error has to be discarded:
// strings.Builder writes cannot fail.
func writeHex(b *strings.Builder, prefix string, r rune, digits int) {
	const hex = "0123456789abcdef"
	b.WriteString(prefix)
	for shift := 4 * (digits - 1); shift >= 0; shift -= 4 {
		b.WriteByte(hex[(r>>shift)&0xf])
	}
}
