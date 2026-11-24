// width.go: how many terminal columns a string occupies.
//
// WHY: alignment needs display width, not bytes or runes. The standard
// library has no width tables, so this covers the two cases that matter for
// names and paths: combining marks take no column, and East Asian wide and
// fullwidth characters (and emoji) take two. Anything rarer may misalign a
// column; it cannot hide or forge content, because Sanitize runs first.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import "unicode"

// wideRanges lists the main double-width blocks.
var wideRanges = [...]struct{ lo, hi rune }{
	{0x1100, 0x115f},   // Hangul Jamo initials
	{0x2e80, 0x303e},   // CJK radicals, punctuation
	{0x3041, 0x33ff},   // Kana, CJK compatibility
	{0x3400, 0x4dbf},   // CJK extension A
	{0x4e00, 0x9fff},   // CJK unified ideographs
	{0xa000, 0xa4cf},   // Yi
	{0xac00, 0xd7a3},   // Hangul syllables
	{0xf900, 0xfaff},   // CJK compatibility ideographs
	{0xfe30, 0xfe4f},   // CJK compatibility forms
	{0xff00, 0xff60},   // fullwidth forms
	{0xffe0, 0xffe6},   // fullwidth signs
	{0x1f300, 0x1f64f}, // pictographs, emoticons
	{0x1f900, 0x1f9ff}, // supplemental pictographs
	{0x20000, 0x3fffd}, // CJK extensions B and later
}

// Width returns the number of terminal columns s occupies. Call it on
// sanitized text: control characters are counted as zero.
func Width(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

func runeWidth(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cc, unicode.Cf) {
		return 0
	}
	for _, w := range wideRanges {
		if r >= w.lo && r <= w.hi {
			return 2
		}
	}
	return 1
}
