// storage_regression_test.go: regression tests for plugin loading and storage errors
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/agilira/orpheus/observe"
)

// A plugin of package orpheus cannot be opened by this test binary, whose
// orpheus package also holds the _test files. The plugin scenarios therefore
// run in a small host program built, with the example memory plugin, in one
// temporary module against the non-test package.
const regressionHostSrc = `package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agilira/orpheus/pkg/orpheus"
)

type nop struct{}

func (nop) Trace(context.Context, string, ...orpheus.Field) {}
func (nop) Debug(context.Context, string, ...orpheus.Field) {}
func (nop) Info(context.Context, string, ...orpheus.Field)  {}
func (nop) Warn(context.Context, string, ...orpheus.Field)  {}
func (nop) Error(context.Context, string, ...orpheus.Field) {}
func (l nop) WithFields(...orpheus.Field) orpheus.Logger    { return l }

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("PANIC:", r)
		}
	}()
	dir := os.Args[2]
	path := filepath.Join(dir, "memory.so")
	cfg := orpheus.DefaultPluginSecurityConfig()
	cfg.AllowedPaths = []string{dir}
	cfg.AllowUnsignedPlugins = true
	ctx := context.Background()
	switch os.Args[1] {
	case "twice":
		pm := orpheus.NewPluginManager(nil, cfg)
		a, err := pm.LoadPlugin(ctx, path)
		if err != nil {
			fmt.Println("ERR", err)
			return
		}
		b, err := pm.LoadPlugin(ctx, path)
		fmt.Println("same:", a == b, "registry:", len(pm.ListLoadedPlugins()), "err:", err)
	case "nochecksum":
		cfg.ValidateChecksums = false
		_, err := orpheus.NewPluginManager(nop{}, cfg).LoadPlugin(ctx, path)
		fmt.Println("err:", err)
	case "config-optin":
		cfg.AllowUnsignedPlugins = false
		pm := orpheus.NewPluginManager(nil, cfg)
		_, err := pm.LoadPluginsFromConfig(ctx, &orpheus.StorageConfig{
			Provider: "memory", PluginPath: path, AllowUnsignedPlugins: true})
		fmt.Println("err:", err)
	}
}
`

// buildRegressionHost builds the host and the plugin into t.TempDir() and
// returns the module directory.
func buildRegressionHost(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Go plugins are not supported on " + runtime.GOOS)
	}
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	prov := filepath.Join(repo, "examples", "storage", "providers")
	writeRegressionModule(t, dir, prov, repo)
	for _, args := range [][]string{
		{"build", "-buildmode=plugin", "-o", "plugins/memory.so", "./plugin"},
		{"build", "-o", "hostbin", "./host"},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// writeRegressionModule lays out go.mod/go.sum from the example providers
// module, the example plugin source and the host source.
func writeRegressionModule(t *testing.T, dir, prov, repo string) {
	t.Helper()
	mod := readFileT(t, filepath.Join(prov, "go.mod"))
	mod = strings.Replace(mod, "module providers", "module regressionhost", 1)
	mod = strings.Replace(mod, "=> ../../..", "=> "+repo, 1)
	files := map[string]string{
		"go.mod":           mod,
		"go.sum":           readFileT(t, filepath.Join(prov, "go.sum")),
		"plugin/memory.go": readFileT(t, filepath.Join(prov, "memory.go")),
		"host/main.go":     regressionHostSrc,
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readFileT(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// runRegressionHost installs the plugin in a fresh symlink-free directory
// and runs one scenario.
func runRegressionHost(t *testing.T, mod, scenario string) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := readFileT(t, filepath.Join(mod, "plugins", "memory.so"))
	if err := os.WriteFile(filepath.Join(dir, "memory.so"), []byte(data), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(filepath.Join(mod, "hostbin"), scenario, dir).CombinedOutput()
	if err != nil {
		t.Fatalf("host %s: %v\n%s", scenario, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestStorageRegression_PluginLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a Go plugin")
	}
	mod := buildRegressionHost(t)

	// The cache used to be looked up by file name but filled by Name(), so
	// it never hit and the constructor ran on every load.
	t.Run("SamePathReturnsCachedPlugin", func(t *testing.T) {
		out := runRegressionHost(t, mod, "twice")
		if out != "same: true registry: 1 err: <nil>" {
			t.Fatalf("second load of the same file: %s", out)
		}
	})
	// The success log sliced hash[:12] although the hash is empty when
	// checksums are disabled.
	t.Run("NoChecksumDoesNotPanic", func(t *testing.T) {
		if out := runRegressionHost(t, mod, "nochecksum"); out != "err: <nil>" {
			t.Fatalf("load with ValidateChecksums=false: %s", out)
		}
	})
	t.Run("StorageConfigOptInLoadsUnsigned", func(t *testing.T) {
		if out := runRegressionHost(t, mod, "config-optin"); out != "err: <nil>" {
			t.Fatalf("StorageConfig.AllowUnsignedPlugins=true: %s", out)
		}
	})
}

// No signature verification exists, so the default must refuse every plugin
// and say how to opt in, before the file is ever opened.
func TestStorageRegression_UnsignedPluginRefusedByDefault(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "memory.so")
	if err := os.WriteFile(path, []byte("not a plugin"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultPluginSecurityConfig()
	cfg.AllowedPaths = []string{dir}
	pm := NewPluginManager(nil, cfg)

	_, err = pm.LoadPlugin(context.Background(), path)
	if !errors.Is(err, errSignatureUnavailable) || !strings.Contains(err.Error(), "AllowUnsignedPlugins") {
		t.Fatalf("LoadPlugin with default config: %v", err)
	}
	_, err = pm.LoadPluginsFromConfig(context.Background(), &Config{Provider: "memory", PluginPath: path})
	if !errors.Is(err, errSignatureUnavailable) {
		t.Fatalf("LoadPluginsFromConfig without opt-in: %v", err)
	}
}

// unsetUserHome clears the environment variables os.UserHomeDir consults
// on Unix, Windows and Plan 9.
func unsetUserHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")
}

// Without a home directory "~/.orpheus/plugins" used to become a directory
// under the working directory, allowed implicitly.
func TestStorageRegression_AllowListNeverUnderWorkingDir(t *testing.T) {
	cwd, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	unsetUserHome(t)
	evil := filepath.Join(cwd, "~", ".orpheus", "plugins", "evil.so")

	if err := NewPluginManager(nil, nil).validatePluginPath(evil); err == nil {
		t.Fatalf("default allow-list accepted %s", evil)
	}
	// An allow-list emptied by expansion must allow nothing.
	only := NewPluginManager(nil, &PluginSecurityConfig{AllowedPaths: []string{"~/.orpheus/plugins"}})
	for _, p := range []string{"/usr/lib/evil.so", filepath.Join(cwd, "evil.so")} {
		if err := only.validatePluginPath(p); err == nil {
			t.Fatalf("allow-list of unexpandable entries accepted %s", p)
		}
	}
}

func TestStorageRegression_DiscoveryCancellationReported(t *testing.T) {
	pm := NewPluginManager(nil, &PluginSecurityConfig{AllowedPaths: []string{t.TempDir()}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pm.DiscoverPlugins(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("DiscoverPlugins with cancelled context: %v", err)
	}
	_, err := pm.LoadPluginsFromConfig(ctx, &Config{Provider: "memory"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadPluginsFromConfig with cancelled context: %v", err)
	}
}

func TestStorageRegression_StorageErrorsKeepCause(t *testing.T) {
	cases := map[string]error{
		"get":         GetError("k", ErrUnavailable),
		"set":         SetError("k", ErrUnavailable),
		"delete":      DeleteError("k", ErrUnavailable),
		"list":        ListError("p", ErrUnavailable),
		"plugin":      PluginLoadError("/p.so", ErrUnavailable),
		"config":      ConfigValidationError("x", ErrUnavailable),
		"unavailable": UnavailableError("x", ErrUnavailable),
	}
	for name, err := range cases {
		if !errors.Is(err, ErrUnavailable) || !IsUnavailable(err) {
			t.Errorf("%s: cause lost: %v", name, err)
		}
	}
	if !IsNotFound(GetError("k", ErrKeyNotFound)) {
		t.Error("IsStorageNotFound(StorageGetError(ErrKeyNotFound)) = false")
	}
}

func TestStorageRegression_IsStorageHelpersSeeWrappedErrors(t *testing.T) {
	wrap := func(err error) error { return fmt.Errorf("loading profile: %w", err) }
	if !IsNotFound(wrap(NotFoundError("profile"))) {
		t.Error("IsStorageNotFound(wrapped) = false")
	}
	if !IsValidationError(wrap(ValidationError("set", "bad"))) {
		t.Error("IsStorageValidationError(wrapped) = false")
	}
	if !IsUnavailable(wrap(UnavailableError("x", errors.New("down")))) {
		t.Error("IsStorageUnavailable(wrapped) = false")
	}
	if IsNotFound(SetError("k", errors.New("disk full"))) {
		t.Error("IsStorageNotFound(unrelated error) = true")
	}
}

// The paths below run in-process without a real plugin: plugin.Open of a Go
// plugin of this package cannot succeed inside its own test binary.

// regressionPluginFile writes a non-plugin file into a fresh allowed
// directory and returns a manager for it plus the file path.
func regressionPluginFile(t *testing.T) (*PluginManager, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "memory.so")
	if err := os.WriteFile(path, []byte("not a plugin"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultPluginSecurityConfig()
	cfg.AllowedPaths = []string{dir}
	return NewPluginManager(&MockLogger{}, cfg), path
}

func TestStorageRegression_RegisterRefusesDuplicateName(t *testing.T) {
	pm := NewPluginManager(nil, nil)
	first := &MockStoragePlugin{name: "memory", version: "1", description: "d"}
	loaded, err := pm.register(first, "/a/memory.so", "/a/memory.so", "abc")
	if err != nil || loaded.Plugin != first || loaded.Metadata["name"] != "memory" {
		t.Fatalf("register: %+v, %v", loaded, err)
	}
	if pm.byPath["/a/memory.so"] != loaded || pm.registry["memory"] != loaded {
		t.Fatal("register did not index by name and by path")
	}
	_, err = pm.register(&MockStoragePlugin{name: "memory"}, "/b/other.so", "/b/other.so", "")
	if err == nil || !strings.Contains(err.Error(), "already loaded from /a/memory.so") {
		t.Fatalf("second file with a loaded name: %v", err)
	}
	if pm.registry["memory"] != loaded || pm.byPath["/b/other.so"] != nil {
		t.Fatal("refused registration changed the registry")
	}
}

func TestStorageRegression_ShortHash(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"abc":              "abc",
		"0123456789ab":     "0123456789ab",
		"0123456789abcdef": "0123456789ab...",
	}
	for in, want := range cases {
		if got := shortHash(in); got != want {
			t.Errorf("shortHash(%q) = %q, want %q", in, got, want)
		}
	}
}

// A file already loaded is returned from the cache before the signature
// refusal, so the default config does not break a repeated load.
func TestStorageRegression_LoadPluginCacheHit(t *testing.T) {
	pm, path := regressionPluginFile(t)
	loaded, err := pm.register(&MockStoragePlugin{name: "memory"}, path, path, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := pm.LoadPlugin(context.Background(), path)
	if err != nil || got != loaded {
		t.Fatalf("LoadPlugin of a loaded file = %p, %v; want cached %p", got, err, loaded)
	}
}

func TestStorageRegression_UnloadForgetsPath(t *testing.T) {
	pm, path := regressionPluginFile(t)
	if _, err := pm.register(&MockStoragePlugin{name: "memory"}, path, path, ""); err != nil {
		t.Fatal(err)
	}
	if err := pm.UnloadPlugin(context.Background(), "memory"); err != nil {
		t.Fatal(err)
	}
	if len(pm.byPath) != 0 {
		t.Fatalf("byPath after unload: %v", pm.byPath)
	}
	// No longer cached, so the load is judged again and refused.
	if _, err := pm.LoadPlugin(context.Background(), path); !errors.Is(err, errSignatureUnavailable) {
		t.Fatalf("LoadPlugin after unload: %v", err)
	}
}

func TestStorageRegression_OpenPluginRejectsNonPlugin(t *testing.T) {
	pm, path := regressionPluginFile(t)
	sp, err := pm.openPlugin(path)
	if sp != nil || err == nil || !strings.Contains(err.Error(), "failed to open plugin") {
		t.Fatalf("openPlugin(non-plugin) = %v, %v", sp, err)
	}
}

// When the working directory is gone, Abs fails and the entry is kept as
// given; a relative entry never matches an absolute plugin path.
func TestStorageRegression_ExpandPluginPathWithoutWorkingDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Skipf("cannot remove the working directory: %v", err)
	}
	if _, err := filepath.Abs("plugins"); err == nil {
		t.Skip("filepath.Abs still works without a working directory here")
	}
	got, ok := expandPluginPath("plugins")
	if !ok || got != "plugins" {
		t.Fatalf("expandPluginPath = %q, %v", got, ok)
	}
	if isWithinDir(got, "/plugins/x.so") {
		t.Fatal("relative allow-list entry matched an absolute path")
	}
}

func TestStorageRegression_LoadPluginHashError(t *testing.T) {
	pm, path := regressionPluginFile(t)
	pm.securityConfig.AllowUnsignedPlugins = true
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("file still readable (running as root?)")
	}
	_, err := pm.LoadPlugin(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "failed to calculate file hash") {
		t.Fatalf("unreadable plugin file: %v", err)
	}
}

// With every allowed entry dropped on expansion there is no path to walk,
// and the cancellation used to be lost before any check ran.
func TestStorageRegression_DiscoveryCancellationWithoutSearchPaths(t *testing.T) {
	unsetUserHome(t)
	pm := NewPluginManager(nil, &PluginSecurityConfig{AllowedPaths: []string{"~/.orpheus/plugins"}})
	if len(pm.pluginPaths) != 0 {
		t.Fatalf("expected no search paths, got %v", pm.pluginPaths)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pm.DiscoverPlugins(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("DiscoverPlugins: %v", err)
	}
	if _, err := pm.LoadPluginsFromConfig(ctx, &Config{Provider: "memory"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadPluginsFromConfig: %v", err)
	}
}

// The caller's cancellation is not a security limit and must not be
// logged as one.
func TestStorageRegression_DiscoveryCancellationNotLoggedAsLimit(t *testing.T) {
	logger := &regressionRecordingLogger{}
	pm := NewPluginManager(logger, &PluginSecurityConfig{AllowedPaths: []string{t.TempDir()}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pm.discoverPluginsInPath(ctx, pm.pluginPaths[0]); !errors.Is(err, context.Canceled) {
		t.Fatalf("discoverPluginsInPath: %v", err)
	}
	for _, msg := range logger.warnings() {
		if strings.Contains(msg, "security limits") {
			t.Fatalf("cancellation logged as %q", msg)
		}
	}
}

type regressionRecordingLogger struct {
	MockLogger
	mu   sync.Mutex
	warn []string
}

func (l *regressionRecordingLogger) Warn(_ context.Context, msg string, _ ...observe.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warn = append(l.warn, msg)
}

func (l *regressionRecordingLogger) warnings() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.warn...)
}
