// access_other.go: current-user access checks where access(2) is
// not available.
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

//go:build !unix

package security

import "os"

// checkCurrentUserAccess keeps the mode-bit answer: without access(2) it is
// the best information available. Windows does not reach this path; its
// read-only attribute is already reflected in the mode bits.
func (fp *FilePermission) checkCurrentUserAccess() {}

// physicalWorkingDir is false: os.Getwd may return a path through links
// (junctions on Windows), so the whole absolute path is walked.
const physicalWorkingDir = false

// workingDir returns the process working directory.
func workingDir() (string, error) {
	return os.Getwd()
}
