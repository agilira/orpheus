// storage_allowed_plugin_paths_test.go: StorageConfig.AllowedPluginPaths is
// appended to the secure plugin-loading defaults, and nothing is added when it
// is empty.
//
// Copyright (c) 2025 AGILira - A. Giordano
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"slices"
	"testing"
)

func TestConfigureStorage_AllowedPluginPathsExtendDefaults(t *testing.T) {
	const custom = "/opt/myapp/plugins"

	app := New("app")
	// The plugin path does not exist, so loading fails and storageErr is set --
	// but the plugin manager is built with the merged security config before
	// the load is attempted, which is what this test inspects.
	app.ConfigureStorage(&StorageConfig{
		Provider:           "memory",
		PluginPath:         "/nonexistent/memory.so",
		AllowedPluginPaths: []string{custom},
	})

	pm := app.PluginManager()
	if pm == nil {
		t.Fatal("ConfigureStorage did not create a plugin manager")
	}

	got := pm.securityConfig.AllowedPaths
	if !slices.Contains(got, custom) {
		t.Errorf("AllowedPluginPaths not honored: %q missing from %v", custom, got)
	}
	for _, def := range DefaultPluginSecurityConfig().AllowedPaths {
		if !slices.Contains(got, def) {
			t.Errorf("default allowed path dropped: %q missing from %v", def, got)
		}
	}
}

func TestConfigureStorage_NoAllowedPluginPathsKeepsDefaults(t *testing.T) {
	app := New("app")
	app.ConfigureStorage(&StorageConfig{
		Provider:   "memory",
		PluginPath: "/nonexistent/memory.so",
	})

	pm := app.PluginManager()
	if pm == nil {
		t.Fatal("ConfigureStorage did not create a plugin manager")
	}

	got := pm.securityConfig.AllowedPaths
	want := DefaultPluginSecurityConfig().AllowedPaths
	if len(got) != len(want) {
		t.Fatalf("expected exactly the default allowed paths %v, got %v", want, got)
	}
	for _, def := range want {
		if !slices.Contains(got, def) {
			t.Errorf("default allowed path dropped: %q missing from %v", def, got)
		}
	}
}
