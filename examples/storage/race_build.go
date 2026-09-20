//go:build race

// race_build.go: reports that this binary was built with the race detector.
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

// raceEnabled is true when the race detector is compiled in.
//
// WHY it exists: a Go plugin can only be opened by a host built the same way,
// and the race detector changes that. The test helper builds memory.so on
// demand, so it must build it with -race exactly when the test binary has it.
const raceEnabled = true
