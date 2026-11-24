// orpheus_fuzz_test.go: fuzz testing of command parsing
//
// The fuzz tests of the security controls, and the rationale behind them,
// are in security/fuzz_test.go.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
)

// =============================================================================
// FUZZ TEST 3: COMMAND PARSING ROBUSTNESS (MEDIUM PRIORITY)
// =============================================================================

// FuzzCommandParsing tests the robustness of command and flag parsing.
//
// SECURITY CRITICALITY: ★★★☆☆ (MEDIUM)
//
// While command parsing is less security-critical than path and input validation,
// it's still important to ensure it never crashes and handles malformed inputs gracefully.
// This prevents denial of service and ensures predictable error messages for users.
//
// ATTACK SURFACE ANALYZED:
// 1. Malformed flag syntax: --flag=, --=value, --, ---flag
// 2. Missing flag values: --flag without argument
// 3. Duplicate flags: --flag value1 --flag value2
// 4. Empty arguments: "", multiple empty strings
// 5. Very long argument lists: 1000+ arguments
// 6. Special flag names: --, -help, --help=--help
//
// SECURITY INVARIANTS:
// 1. NEVER panic on any argument array
// 2. NEVER cause infinite loops or hangs
// 3. Provide clear error messages for malformed input
// 4. Handle empty args array gracefully
// 5. Reject obviously malformed flag syntax consistently
func FuzzCommandParsing(f *testing.F) {
	// Seed corpus with valid command patterns
	f.Add("start")
	f.Add("stop")
	f.Add("--flag value")
	f.Add("-f value")
	f.Add("command --flag=value")
	f.Add("cmd sub1 sub2 --flag")

	// Seed corpus with malformed patterns
	f.Add("")
	f.Add("--")
	f.Add("---")
	f.Add("--flag=")
	f.Add("--=value")
	f.Add("-" + strings.Repeat("a", 100))

	f.Fuzz(func(t *testing.T, argsStr string) {
		// Create a simple test app
		app := New("testapp")
		app.Command("test", "Test command", func(ctx *Context) error {
			return nil
		})

		// Parse the fuzzed string into arguments
		// Simple space-splitting for fuzzing purposes
		args := strings.Fields(argsStr)

		// CRITICAL: Must never panic
		// If it panics, the fuzzer will catch it as a failure
		_ = app.Run(args)

		// The function should handle any input gracefully
		// We don't check for specific errors, just that it doesn't crash
	})
}

// =============================================================================
// HELPER FUNCTIONS FOR FUZZ TESTING
// =============================================================================

// TODO: Add more helper functions if needed for corpus generation or validation
