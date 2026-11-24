// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"testing"

	goerrors "github.com/agilira/go-errors"

	"github.com/agilira/orpheus/errs"
	"github.com/agilira/orpheus/internal/apicheck"
	"github.com/agilira/orpheus/pkg/orpheus"
)

// mirrored are the packages whose API this facade re-exports.
var mirrored = []string{
	apicheck.Module + "/errs",
	apicheck.Module + "/observe",
}

// facadeNames maps an identifier of a mirrored package to its v1.11.0 name
// when the two differ.
var facadeNames = map[string]string{}

// notMirrored lists identifiers added to the new packages after v1.11.0;
// the facade keeps the v1.11.0 API and leaves them out.
var notMirrored = map[string]bool{
	"errs.NewErrorWithCause": true,
}

// TestFacadeMirrorsPackages fails when a mirrored package exports something
// the facade does not re-export as the very same type, constant, variable
// or function signature.
func TestFacadeMirrorsPackages(t *testing.T) {
	u, err := apicheck.Load(mirrored...)
	if err != nil {
		t.Fatal(err)
	}
	problems, err := u.Mirror(apicheck.Module+"/pkg/orpheus", mirrored, facadeNames, notMirrored)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// TestLegacyErrorNames covers the two names kept only by this facade.
func TestLegacyErrorNames(t *testing.T) {
	var e *orpheus.OrpheusError = orpheus.NewOrpheusError(orpheus.ErrCodeValidation, "cmd", "msg")
	var _ *errs.Error = e
	if e.ErrorCode() != goerrors.ErrorCode("ORF1000") || e.Command != "cmd" {
		t.Errorf("NewOrpheusError: got %v", e)
	}
}
