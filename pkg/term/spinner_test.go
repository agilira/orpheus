// spinner_test.go: tests for the delayed progress spinner.
//
// WHY: a spinner shares the terminal with the report that follows it. It
// must stay invisible for fast operations and outside terminals, erase
// itself completely, never print raw bytes from its label, and survive
// concurrent progress updates from the goroutine doing the work.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agilira/orpheus/pkg/term"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

var fast = term.SpinnerOptions{Delay: time.Millisecond, Interval: time.Millisecond}

func waitFor(t *testing.T, b *syncBuffer, substr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(b.String(), substr) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %q in %q", substr, b.String())
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSpinnerSilentWithoutAnimation(t *testing.T) {
	var b syncBuffer
	s := term.StartSpinner(&b, term.Caps{Color: true}, "hashing", fast)
	s.Update(1, 2)
	time.Sleep(20 * time.Millisecond)
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if b.String() != "" {
		t.Fatalf("spinner wrote without animation: %q", b.String())
	}
}

func TestSpinnerSilentForFastOperations(t *testing.T) {
	var b syncBuffer
	s := term.StartSpinner(&b, colorCaps, "hashing", term.SpinnerOptions{Delay: time.Hour})
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if b.String() != "" {
		t.Fatalf("spinner wrote before its delay: %q", b.String())
	}
}

func TestSpinnerShowsProgressAndClears(t *testing.T) {
	var b syncBuffer
	s := term.StartSpinner(&b, colorCaps, "model.gguf", fast)
	s.Update(500_000_000, 2_000_000_000)
	waitFor(t, &b, "model.gguf 500.0 MB / 2.0 GB (25%)")
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(b.String(), "\r\x1b[2K") {
		t.Fatalf("spinner did not clear its line: %q", b.String())
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}

func TestSpinnerUnknownTotal(t *testing.T) {
	var b syncBuffer
	s := term.StartSpinner(&b, colorCaps, "tree", fast)
	s.Update(1500, 0)
	waitFor(t, &b, "tree 1.5 kB")
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "%") {
		t.Fatalf("percentage shown without a total: %q", b.String())
	}
}

func TestSpinnerSanitizesAndBoundsLabel(t *testing.T) {
	var b syncBuffer
	label := "evil\x1b[31m\n" + strings.Repeat("x", 500)
	s := term.StartSpinner(&b, colorCaps, label, fast)
	waitFor(t, &b, "...")
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, "\n") {
		t.Fatalf("label broke the line: %q", out)
	}
	for _, frame := range strings.Split(out, "\r\x1b[2K") {
		if strings.Contains(frame, "\x1b") {
			t.Fatalf("raw escape in frame %q", frame)
		}
		if w := term.Width(frame); w > 80 {
			t.Fatalf("frame is %d columns wide: %q", w, frame)
		}
	}
}

func TestSpinnerReturnsWriteError(t *testing.T) {
	s := term.StartSpinner(failWriter{}, colorCaps, "x", fast)
	time.Sleep(20 * time.Millisecond)
	if err := s.Stop(); err == nil {
		t.Fatal("Stop swallowed the write error")
	}
}

func TestSpinnerConcurrentUse(t *testing.T) {
	var b syncBuffer
	s := term.StartSpinner(&b, colorCaps, "x", fast)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 200 {
				s.Update(int64(i*j), 1600)
			}
		}()
	}
	wg.Wait()
	errs := make(chan error, 2)
	for range 2 {
		go func() { errs <- s.Stop() }()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	s.Update(1, 1)
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{-5, "0 B"},
		{0, "0 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{2_400_000_000, "2.4 GB"},
		{1<<63 - 1, "9.2 EB"},
	}
	for _, tc := range cases {
		if got := term.FormatBytes(tc.n); got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func FuzzSpinnerProgress(f *testing.F) {
	f.Add("x", int64(1), int64(2))
	f.Add("\x1b]0;t\x07", int64(-1), int64(-1))
	f.Add("模型", int64(1<<62), int64(1))
	f.Fuzz(func(t *testing.T, label string, done, total int64) {
		line := term.ProgressLine(label, done, total)
		if strings.ContainsAny(line, "\x1b\r\n") {
			t.Fatalf("ProgressLine(%q) = %q contains a control", label, line)
		}
		if w := term.Width(line); w > 76 {
			t.Fatalf("ProgressLine(%q) is %d columns", label, w)
		}
		if strings.HasSuffix(line, "%)") && total <= 0 {
			t.Fatalf("percentage without a total: %q", line)
		}
		if strings.HasSuffix(line, "%)") && strings.Contains(line[strings.LastIndex(line, "("):], "-") {
			t.Fatalf("negative percentage: %q", line)
		}
	})
}
