// detect.go: decide whether output may carry colors and animation.
//
// WHY: styling is only useful to a human looking at a terminal. Written to a
// pipe, a file or a CI log it becomes noise that breaks grep and parsers, so
// detection fails closed: unless the writer is clearly a terminal, output is
// plain. Only the standard library is used, which means a character device
// counts as a terminal; /dev/null is one too, and losing styling there is
// harmless.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import (
	"io"
	"io/fs"
	"os"
)

// Caps describes what a writer can display.
type Caps struct {
	// Color allows ANSI colors and text attributes.
	Color bool
	// Animate allows in-place updates such as a spinner.
	Animate bool
}

// statter is satisfied by *os.File and by test doubles.
type statter interface {
	Stat() (fs.FileInfo, error)
}

// Detect reports the capabilities of w. lookupEnv reads the environment;
// nil means os.LookupEnv. noColor is the value of a --no-color flag.
//
// Color follows https://no-color.org: a non-empty NO_COLOR disables it.
// TERM=dumb disables both color and animation.
func Detect(w io.Writer, lookupEnv func(string) (string, bool), noColor bool) Caps {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if !isTerminal(w) || envValue(lookupEnv, "TERM") == "dumb" {
		return Caps{}
	}
	color := !noColor && envValue(lookupEnv, "NO_COLOR") == ""
	return Caps{Color: color, Animate: true}
}

func isTerminal(w io.Writer) bool {
	s, ok := w.(statter)
	if !ok {
		return false
	}
	info, err := s.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&fs.ModeCharDevice != 0
}

func envValue(lookupEnv func(string) (string, bool), key string) string {
	v, _ := lookupEnv(key)
	return v
}
