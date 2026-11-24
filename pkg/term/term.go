// Package term keeps the terminal helpers at their original import path.
// The same API is available at github.com/agilira/orpheus/term; the names
// here are aliases of its types and wrappers of its functions.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package term

import (
	"io"

	"github.com/agilira/orpheus/term"
)

// Types, see package github.com/agilira/orpheus/term.
type (
	Caps           = term.Caps
	Cell           = term.Cell
	Spinner        = term.Spinner
	SpinnerOptions = term.SpinnerOptions
	Style          = term.Style
	Table          = term.Table
)

// Styles, see package github.com/agilira/orpheus/term.
const (
	None   = term.None
	Green  = term.Green
	Yellow = term.Yellow
	Red    = term.Red
	Gray   = term.Gray
	Bold   = term.Bold
)

// Detect calls term.Detect.
func Detect(w io.Writer, lookupEnv func(string) (string, bool), noColor bool) Caps {
	return term.Detect(w, lookupEnv, noColor)
}

// FormatBytes calls term.FormatBytes.
func FormatBytes(n int64) string { return term.FormatBytes(n) }

// IsSafeRune calls term.IsSafeRune.
func IsSafeRune(r rune) bool { return term.IsSafeRune(r) }

// NewTable calls term.NewTable.
func NewTable(caps Caps, headers ...string) Table { return term.NewTable(caps, headers...) }

// Paint calls term.Paint.
func Paint(caps Caps, style Style, text string) string { return term.Paint(caps, style, text) }

// ProgressLine calls term.ProgressLine.
func ProgressLine(label string, done, total int64) string {
	return term.ProgressLine(label, done, total)
}

// Sanitize calls term.Sanitize.
func Sanitize(s string) string { return term.Sanitize(s) }

// StartSpinner calls term.StartSpinner.
func StartSpinner(w io.Writer, caps Caps, label string, opts SpinnerOptions) Spinner {
	return term.StartSpinner(w, caps, label, opts)
}

// Styled calls term.Styled.
func Styled(style Style, s string) Cell { return term.Styled(style, s) }

// Text calls term.Text.
func Text(s string) Cell { return term.Text(s) }

// Width calls term.Width.
func Width(s string) int { return term.Width(s) }
