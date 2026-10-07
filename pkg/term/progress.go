// progress.go: the text a spinner shows.
//
// WHY: kept apart from the spinner goroutine so that the part handling
// untrusted input (the label) and arithmetic on caller-supplied numbers is a
// pure function that can be fuzzed directly.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import (
	"strconv"
	"strings"
)

// maxLabelWidth keeps a progress line on one row of an 80-column terminal;
// a wrapped line could not be erased with a carriage return.
const maxLabelWidth = 40

const ellipsis = "..."

var byteUnits = [...]string{"kB", "MB", "GB", "TB", "PB", "EB"}

// FormatBytes renders n in decimal units with one decimal place.
// Negative values render as zero.
func FormatBytes(n int64) string {
	if n < 1000 {
		return strconv.FormatInt(max(n, 0), 10) + " B"
	}
	v := float64(n) / 1000
	unit := 0
	for v >= 1000 && unit < len(byteUnits)-1 {
		v /= 1000
		unit++
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + " " + byteUnits[unit]
}

// ProgressLine returns the sanitized, width-bounded label followed by the
// progress: "label 1.2 GB / 2.4 GB (50%)" with a known total, "label 1.2 GB"
// without one, and the label alone when nothing is known yet.
func ProgressLine(label string, done, total int64) string {
	line := truncate(Sanitize(label), maxLabelWidth)
	switch {
	case total > 0:
		return line + " " + FormatBytes(done) + " / " + FormatBytes(total) +
			" (" + strconv.Itoa(percent(done, total)) + "%)"
	case done > 0:
		return line + " " + FormatBytes(done)
	default:
		return line
	}
}

func percent(done, total int64) int {
	if done <= 0 {
		return 0
	}
	if done >= total {
		return 100
	}
	// Float division avoids overflowing done*100 near the int64 limit.
	return int(float64(done) / float64(total) * 100)
}

func truncate(s string, width int) string {
	if Width(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := runeWidth(r)
		if used+w > width-len(ellipsis) {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String() + ellipsis
}
