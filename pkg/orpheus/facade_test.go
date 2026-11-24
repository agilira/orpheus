// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"errors"
	"fmt"
	"testing"

	goerrors "github.com/agilira/go-errors"

	"github.com/agilira/orpheus/errs"
	"github.com/agilira/orpheus/internal/apicheck"
	"github.com/agilira/orpheus/pkg/orpheus"
	"github.com/agilira/orpheus/storage"
)

// mirrored are the packages whose API this facade re-exports.
var mirrored = []string{
	apicheck.Module + "/errs",
	apicheck.Module + "/observe",
	apicheck.Module + "/storage",
	apicheck.Module + "/security",
	apicheck.Module + "/prompt",
}

// facadeNames maps an identifier of a mirrored package to its v1.11.0 name
// when the two differ.
var facadeNames = map[string]string{
	"prompt.Terminal":           "TerminalPrompter",
	"prompt.NewTerminal":        "NewTerminalPrompter",
	"prompt.NewFrom":            "NewPrompterFrom",
	"security.Config":           "SecurityConfig",
	"security.DefaultConfig":    "DefaultSecurityConfig",
	"storage.Config":            "StorageConfig",
	"storage.Stats":             "StorageStats",
	"storage.Plugin":            "StoragePlugin",
	"storage.GetError":          "StorageGetError",
	"storage.SetError":          "StorageSetError",
	"storage.DeleteError":       "StorageDeleteError",
	"storage.ListError":         "StorageListError",
	"storage.NotFoundError":     "StorageNotFoundError",
	"storage.ValidationError":   "StorageValidationError",
	"storage.UnavailableError":  "StorageUnavailableError",
	"storage.IsNotFound":        "IsStorageNotFound",
	"storage.IsValidationError": "IsStorageValidationError",
	"storage.IsUnavailable":     "IsStorageUnavailable",
	"storage.ErrNotConfigured":  "ErrStorageNotConfigured",
	"storage.ErrUnavailable":    "ErrStorageUnavailable",
	"storage.ErrClosed":         "ErrStorageClosed",
}

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

// TestSentinelsAreTheSameValues checks that every error variable of the
// facade is the value of the package that defines it, so errors.Is matches
// across the old and the new name.
func TestSentinelsAreTheSameValues(t *testing.T) {
	pairs := map[string][2]error{
		"ErrKeyNotFound":          {orpheus.ErrKeyNotFound, storage.ErrKeyNotFound},
		"ErrKeyEmpty":             {orpheus.ErrKeyEmpty, storage.ErrKeyEmpty},
		"ErrKeyTooLong":           {orpheus.ErrKeyTooLong, storage.ErrKeyTooLong},
		"ErrKeyInvalid":           {orpheus.ErrKeyInvalid, storage.ErrKeyInvalid},
		"ErrValueTooLarge":        {orpheus.ErrValueTooLarge, storage.ErrValueTooLarge},
		"ErrStorageNotConfigured": {orpheus.ErrStorageNotConfigured, storage.ErrNotConfigured},
		"ErrStorageUnavailable":   {orpheus.ErrStorageUnavailable, storage.ErrUnavailable},
		"ErrStorageClosed":        {orpheus.ErrStorageClosed, storage.ErrClosed},
		"ErrPluginNotFound":       {orpheus.ErrPluginNotFound, storage.ErrPluginNotFound},
		"ErrPluginInvalid":        {orpheus.ErrPluginInvalid, storage.ErrPluginInvalid},
		"ErrConfigInvalid":        {orpheus.ErrConfigInvalid, storage.ErrConfigInvalid},
	}
	for name, p := range pairs {
		if p[0] != p[1] || !errors.Is(fmt.Errorf("wrapped: %w", p[1]), p[0]) {
			t.Errorf("%s is not the value of package storage", name)
		}
	}
}
