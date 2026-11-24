// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"flag"
	"testing"

	"github.com/agilira/orpheus/internal/apicheck"
)

var updateAPI = flag.Bool("update-api", false, "rewrite testdata/api_v1.11.0.txt")

// TestAPIUnchanged fails when the exported API of this package differs from
// the v1.11.0 API recorded in testdata/api_v1.11.0.txt: a missing or extra
// identifier, a changed type, signature, constant value or method set.
func TestAPIUnchanged(t *testing.T) {
	u, err := apicheck.Load()
	if err != nil {
		t.Fatal(err)
	}
	diff, err := u.Diff(apicheck.Module+"/pkg/term", "testdata/api_v1.11.0.txt", *updateAPI)
	if err != nil {
		t.Fatal(err)
	}
	if diff != "" {
		t.Errorf("API differs from v1.11.0 (- missing, + extra):\n%s", diff)
	}
}
