//go:build !race

// norace_build.go: reports that this binary was built without the race detector.
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package main

// raceEnabled is false when the race detector is not compiled in. See
// race_build.go for why the distinction matters.
const raceEnabled = false
