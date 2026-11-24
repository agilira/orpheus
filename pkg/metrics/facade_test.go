// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics_test

import (
	"testing"

	"github.com/agilira/orpheus/internal/apicheck"
)

// TestFacadeMirrorsPackage fails when github.com/agilira/orpheus/metrics
// exports something this facade does not re-export as the very same type,
// constant or function signature.
func TestFacadeMirrorsPackage(t *testing.T) {
	src := apicheck.Module + "/metrics"
	u, err := apicheck.Load(src)
	if err != nil {
		t.Fatal(err)
	}
	problems, err := u.Mirror(apicheck.Module+"/pkg/metrics", []string{src}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}
