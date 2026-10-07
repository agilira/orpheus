// storage_plugin_match_test.go: which plugin file a configuration may load.
//
// WHY: loading a Go plugin runs its code in-process, and a plugin built with
// another Go version kills the process with an unrecoverable fatal error.
// Choosing the file is therefore a security decision. These tests pin three
// rules: the provider name must match a plugin file exactly, an allowed
// directory is a directory boundary and not a string prefix, and no default
// search path depends on the current working directory.
//
// The fixtures are not valid plugins, so any test that reaches plugin.Open
// sees an ordinary error instead of loading code.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pluginDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("not a plugin"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func managerFor(dirs ...string) *PluginManager {
	config := DefaultPluginSecurityConfig()
	config.AllowedPaths = dirs
	config.ValidateChecksums = false
	return NewPluginManager(&MockLogger{}, config)
}

func TestLoadPluginsFromConfigRejectsEmptyProvider(t *testing.T) {
	pm := managerFor(pluginDir(t, "memory.so"))
	_, err := pm.LoadPluginsFromConfig(context.Background(), &StorageConfig{Provider: ""})
	if err == nil {
		t.Fatal("empty provider was accepted")
	}
	if strings.Contains(err.Error(), "failed to open plugin") {
		t.Fatalf("empty provider reached plugin.Open: %v", err)
	}
}

func TestLoadPluginsFromConfigRejectsPartialName(t *testing.T) {
	for _, provider := range []string{"mem", "emo", "y", ".so", "memory.so"} {
		t.Run(provider, func(t *testing.T) {
			pm := managerFor(pluginDir(t, "memory.so"))
			_, err := pm.LoadPluginsFromConfig(context.Background(), &StorageConfig{Provider: provider})
			if err == nil || !strings.Contains(err.Error(), "no plugin found") {
				t.Fatalf("provider %q selected a plugin it does not name: %v", provider, err)
			}
		})
	}
}

func TestLoadPluginsFromConfigSelectsExactName(t *testing.T) {
	dir := pluginDir(t, "memory-evil.so", "memory.so")
	pm := managerFor(dir)
	_, err := pm.LoadPluginsFromConfig(context.Background(), &StorageConfig{Provider: "memory"})
	if err == nil {
		t.Fatal("a fixture that is not a plugin was loaded")
	}
	want := filepath.Join(dir, "memory.so")
	if !strings.Contains(err.Error(), "failed to open plugin") || !strings.Contains(err.Error(), want) {
		t.Fatalf("provider memory did not select %s: %v", want, err)
	}
}

func TestValidatePluginPathRespectsDirectoryBoundary(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "plugins")
	pm := managerFor(allowed)
	rejected := []string{
		filepath.Join(root, "plugins-evil", "x.so"),
		allowed + string(filepath.Separator) + ".." + string(filepath.Separator) + "x.so",
		allowed + "x.so",
		allowed,
	}
	for _, p := range rejected {
		if err := pm.validatePluginPath(p); err == nil {
			t.Errorf("validatePluginPath(%q) accepted a path outside %q", p, allowed)
		}
	}
	accepted := []string{
		filepath.Join(allowed, "x.so"),
		filepath.Join(allowed, "sub", "x.so"),
	}
	for _, p := range accepted {
		if err := pm.validatePluginPath(p); err != nil {
			t.Errorf("validatePluginPath(%q) = %v, want accepted", p, err)
		}
	}
}

func TestDefaultPluginPathsDoNotDependOnWorkingDirectory(t *testing.T) {
	for _, p := range DefaultPluginSecurityConfig().AllowedPaths {
		if !filepath.IsAbs(p) && !strings.HasPrefix(p, "~/") {
			t.Errorf("default plugin path %q is relative to the working directory", p)
		}
	}
}
