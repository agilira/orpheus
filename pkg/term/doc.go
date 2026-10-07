// Package term provides terminal output helpers for Orpheus applications:
// capability detection, sanitized colored text, aligned tables and a
// delayed progress spinner.
//
// Every function that prints text sanitizes it first, because CLI output
// routinely contains names chosen by someone else. Detection fails closed:
// output that is not clearly going to a terminal is plain text.
//
// The package depends only on the standard library and is optional:
// applications that do not import it pay nothing for it.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package term
