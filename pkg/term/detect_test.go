// detect_test.go: tests for terminal capability detection.
//
// WHY: colors and spinners written into a pipe, a log file or a CI job
// become garbage that other tools must parse. Detection must fail closed:
// anything that is not clearly an interactive terminal gets plain text.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agilira/orpheus/pkg/term"
)

type fakeInfo struct{ mode fs.FileMode }

func (f fakeInfo) Name() string       { return "fake" }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() any           { return nil }

type fakeFile struct {
	bytes.Buffer
	mode fs.FileMode
	err  error
}

func (f *fakeFile) Stat() (fs.FileInfo, error) { return fakeInfo{f.mode}, f.err }

func tty() *fakeFile { return &fakeFile{mode: fs.ModeDevice | fs.ModeCharDevice} }

func env(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := vars[k]
		return v, ok
	}
}

func TestDetectInteractiveTerminal(t *testing.T) {
	c := term.Detect(tty(), env(map[string]string{"TERM": "xterm-256color"}), false)
	if !c.Color || !c.Animate {
		t.Fatalf("Detect on a TTY = %+v, want color and animation", c)
	}
}

func TestDetectFailsClosed(t *testing.T) {
	regular := &fakeFile{mode: 0o644}
	statErr := &fakeFile{mode: fs.ModeCharDevice, err: errors.New("boom")}
	cases := []struct {
		name string
		w    io.Writer
	}{
		{"plain writer without Stat", &bytes.Buffer{}},
		{"regular file", regular},
		{"Stat error", statErr},
		{"nil writer", nil},
		{"typed nil *os.File", (*os.File)(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := term.Detect(tc.w, env(nil), false)
			if c.Color || c.Animate {
				t.Fatalf("Detect = %+v, want plain output", c)
			}
		})
	}
}

func TestDetectRealPipeIsPlain(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	if c := term.Detect(w, env(nil), false); c.Color || c.Animate {
		t.Fatalf("Detect on a pipe = %+v, want plain output", c)
	}
}

func TestDetectRealFileIsPlain(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "out.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	if c := term.Detect(f, env(nil), false); c.Color || c.Animate {
		t.Fatalf("Detect on a file = %+v, want plain output", c)
	}
}

func TestDetectNoColor(t *testing.T) {
	cases := []struct {
		name      string
		vars      map[string]string
		flag      bool
		wantColor bool
	}{
		{"NO_COLOR set", map[string]string{"NO_COLOR": "1"}, false, false},
		{"NO_COLOR any value", map[string]string{"NO_COLOR": "false"}, false, false},
		{"NO_COLOR empty is ignored", map[string]string{"NO_COLOR": ""}, false, true},
		{"--no-color flag", nil, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := term.Detect(tty(), env(tc.vars), tc.flag)
			if c.Color != tc.wantColor {
				t.Fatalf("Color = %v, want %v", c.Color, tc.wantColor)
			}
			if !c.Animate {
				t.Fatal("disabling color must not disable the spinner")
			}
		})
	}
}

func TestDetectDumbTerminal(t *testing.T) {
	c := term.Detect(tty(), env(map[string]string{"TERM": "dumb"}), false)
	if c.Color || c.Animate {
		t.Fatalf("Detect with TERM=dumb = %+v, want plain output", c)
	}
}

func TestDetectNilEnvUsesProcessEnvironment(t *testing.T) {
	t.Setenv("TERM", "xterm")
	t.Setenv("NO_COLOR", "1")
	if c := term.Detect(tty(), nil, false); c.Color || !c.Animate {
		t.Fatalf("Detect with nil env = %+v, want NO_COLOR from the process", c)
	}
	t.Setenv("NO_COLOR", "")
	if c := term.Detect(tty(), nil, false); !c.Color || !c.Animate {
		t.Fatalf("Detect with nil env = %+v, want defaults for a TTY", c)
	}
}
