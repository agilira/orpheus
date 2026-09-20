// storage_config_error_test.go: ConfigureStorage keeps the application
// running when storage cannot be set up, but the reason must be reachable.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"strings"
	"testing"
)

// A provider that cannot be loaded must leave a reason behind, with no
// logger configured.
func TestConfigureStorageRecordsLoadFailure(t *testing.T) {
	app := New("myapp").ConfigureStorage(&StorageConfig{
		Provider: "no-such-provider",
		Config:   map[string]interface{}{},
	})

	err := app.StorageError()
	if err == nil {
		t.Fatal("StorageError() = nil: the failure was swallowed with no logger to see it")
	}
	if !strings.Contains(err.Error(), "no-such-provider") {
		t.Errorf("StorageError() = %v, want the provider named", err)
	}
	if app.Storage() != nil {
		t.Error("Storage() is set although configuration failed")
	}
}

// A nil configuration is a failure to configure, not a success.
func TestConfigureStorageRecordsNilConfig(t *testing.T) {
	app := New("myapp").ConfigureStorage(nil)

	if app.StorageError() == nil {
		t.Fatal("StorageError() = nil after ConfigureStorage(nil)")
	}
}

// Nothing configured at all is not an error.
func TestStorageErrorIsNilWhenNeverConfigured(t *testing.T) {
	if err := New("myapp").StorageError(); err != nil {
		t.Fatalf("StorageError() = %v, want nil", err)
	}
}
