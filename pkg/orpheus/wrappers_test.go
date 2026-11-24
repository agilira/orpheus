// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	root "github.com/agilira/orpheus"
	"github.com/agilira/orpheus/errs"
	"github.com/agilira/orpheus/observe"
	"github.com/agilira/orpheus/pkg/orpheus"
	"github.com/agilira/orpheus/prompt"
	"github.com/agilira/orpheus/security"
	"github.com/agilira/orpheus/storage"
)

// sameError fails unless a and b carry the same message, code, user
// message, exit code and cause.
func sameError(t *testing.T, name string, a, b *errs.Error) {
	t.Helper()
	if a.Error() != b.Error() || a.ErrorCode() != b.ErrorCode() || a.UserMessage() != b.UserMessage() ||
		a.ExitCode() != b.ExitCode() || a.IsRetryable() != b.IsRetryable() {
		t.Errorf("%s: facade %q (%s) differs from %q (%s)", name, a, a.ErrorCode(), b, b.ErrorCode())
	}
}

// TestWrappersCallTheirTarget checks that every function of the facade
// returns what the function it wraps returns: a wrapper calling the wrong
// target would still have the right signature.
func TestWrappersCallTheirTarget(t *testing.T) {
	cause := errors.New("cause")

	sameError(t, "NewError", orpheus.NewError(errs.ErrCodeExecution, "c", "m"), errs.NewError(errs.ErrCodeExecution, "c", "m"))
	sameError(t, "NewOrpheusError", orpheus.NewOrpheusError(errs.ErrCodeExecution, "c", "m"), errs.NewError(errs.ErrCodeExecution, "c", "m"))
	sameError(t, "ValidationError", orpheus.ValidationError("c", "m"), errs.ValidationError("c", "m"))
	sameError(t, "ExecutionError", orpheus.ExecutionError("c", "m"), errs.ExecutionError("c", "m"))
	sameError(t, "NotFoundError", orpheus.NotFoundError("c", "m"), errs.NotFoundError("c", "m"))
	sameError(t, "InternalError", orpheus.InternalError("m"), errs.InternalError("m"))

	sameError(t, "StorageGetError", orpheus.StorageGetError("k", cause), storage.GetError("k", cause))
	sameError(t, "StorageSetError", orpheus.StorageSetError("k", cause), storage.SetError("k", cause))
	sameError(t, "StorageDeleteError", orpheus.StorageDeleteError("k", cause), storage.DeleteError("k", cause))
	sameError(t, "StorageListError", orpheus.StorageListError("p", cause), storage.ListError("p", cause))
	sameError(t, "StorageNotFoundError", orpheus.StorageNotFoundError("k"), storage.NotFoundError("k"))
	sameError(t, "StorageValidationError", orpheus.StorageValidationError("op", "m"), storage.ValidationError("op", "m"))
	sameError(t, "StorageUnavailableError", orpheus.StorageUnavailableError("p", cause), storage.UnavailableError("p", cause))
	sameError(t, "PluginLoadError", orpheus.PluginLoadError("x.so", cause), storage.PluginLoadError("x.so", cause))
	sameError(t, "ConfigValidationError", orpheus.ConfigValidationError("p", cause), storage.ConfigValidationError("p", cause))
	if !errors.Is(orpheus.StorageGetError("k", cause), cause) {
		t.Error("StorageGetError lost its cause")
	}

	for _, err := range []error{orpheus.StorageNotFoundError("k"), orpheus.StorageValidationError("op", "m"),
		orpheus.StorageUnavailableError("p", cause), cause} {
		if orpheus.IsStorageNotFound(err) != storage.IsNotFound(err) ||
			orpheus.IsStorageValidationError(err) != storage.IsValidationError(err) ||
			orpheus.IsStorageUnavailable(err) != storage.IsUnavailable(err) {
			t.Errorf("IsStorage* disagree with package storage on %v", err)
		}
	}
	if !orpheus.IsStorageNotFound(orpheus.StorageNotFoundError("k")) || !orpheus.IsStorageValidationError(orpheus.StorageValidationError("o", "m")) ||
		!orpheus.IsStorageUnavailable(orpheus.StorageUnavailableError("p", cause)) {
		t.Error("IsStorage* do not recognise their own errors")
	}

	fields := map[string][2]observe.Field{
		"StringField":  {orpheus.StringField("k", "v"), observe.StringField("k", "v")},
		"IntField":     {orpheus.IntField("k", 1), observe.IntField("k", 1)},
		"Float64Field": {orpheus.Float64Field("k", 1.5), observe.Float64Field("k", 1.5)},
		"BoolField":    {orpheus.BoolField("k", true), observe.BoolField("k", true)},
		"ErrorField":   {orpheus.ErrorField(cause), observe.ErrorField(cause)},
	}
	for name, f := range fields {
		if f[0] != f[1] {
			t.Errorf("%s: %+v != %+v", name, f[0], f[1])
		}
	}

	if !reflect.DeepEqual(orpheus.DefaultPluginSecurityConfig(), storage.DefaultPluginSecurityConfig()) {
		t.Error("DefaultPluginSecurityConfig differs")
	}
	if orpheus.NewPluginManager(nil, nil) == nil {
		t.Error("NewPluginManager returned nil")
	}
	if !reflect.DeepEqual(orpheus.DefaultSecurityConfig(), security.DefaultConfig()) {
		t.Error("DefaultSecurityConfig differs")
	}
	if !reflect.DeepEqual(orpheus.DefaultValidationConfig(), security.DefaultValidationConfig()) {
		t.Error("DefaultValidationConfig differs")
	}
	if orpheus.NewInputValidator(orpheus.DefaultValidationConfig()).ValidateStringFlag("name", "a;b").IsValid {
		t.Error("NewInputValidator: validator accepted a shell separator")
	}

	cfg := orpheus.DefaultSecurityConfig()
	for _, p := range []string{"data/x.json", "../../etc/passwd"} {
		if !reflect.DeepEqual(orpheus.ValidateSecurePath(p, cfg), security.ValidateSecurePath(p, cfg)) {
			t.Errorf("ValidateSecurePath(%q) differs", p)
		}
	}
	file := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	a1, e1 := orpheus.AnalyzeFilePermissions(file, cfg)
	a2, e2 := security.AnalyzeFilePermissions(file, cfg)
	if !reflect.DeepEqual(a1, a2) || (e1 == nil) != (e2 == nil) {
		t.Error("AnalyzeFilePermissions differs")
	}
	checks := []struct {
		name   string
		f, g   func(string, security.Config) (bool, error)
		expect bool
	}{
		{"CheckFileReadable", orpheus.CheckFileReadable, security.CheckFileReadable, true},
		{"CheckFileWritable", orpheus.CheckFileWritable, security.CheckFileWritable, true},
		{"CheckFileExecutable", orpheus.CheckFileExecutable, security.CheckFileExecutable, false},
	}
	for _, c := range checks {
		x, _ := c.f(file, cfg)
		y, _ := c.g(file, cfg)
		if x != y || x != c.expect {
			t.Errorf("%s: facade %v, security %v, want %v", c.name, x, y, c.expect)
		}
	}

	var out1, out2 bytes.Buffer
	v1, _ := orpheus.NewPrompterFrom(strings.NewReader("yes\n"), &out1, -1).Ask("q", "")
	v2, _ := prompt.NewFrom(strings.NewReader("yes\n"), &out2, -1).Ask("q", "")
	if v1 != v2 || out1.String() != out2.String() {
		t.Errorf("NewPrompterFrom: %q %q differs from %q %q", v1, out1.String(), v2, out2.String())
	}
	if orpheus.NewTerminalPrompter() == nil {
		t.Error("NewTerminalPrompter returned nil")
	}

	if orpheus.New("app").GenerateHelp() != root.New("app").GenerateHelp() {
		t.Error("New differs")
	}
	if c := orpheus.NewCommand("c", "d"); c.Name() != "c" || c.Description() != "d" {
		t.Error("NewCommand differs")
	}
	app := orpheus.New("app")
	if orpheus.NewHelpGenerator(app).GenerateAppHelp() != root.NewHelpGenerator(app).GenerateAppHelp() {
		t.Error("NewHelpGenerator differs")
	}
}
