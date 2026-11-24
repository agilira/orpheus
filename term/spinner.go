// spinner.go: a progress indicator for slow operations.
//
// WHY: hashing a multi-gigabyte model takes seconds and a silent terminal
// looks hung. Fast operations must not flicker, so nothing is drawn before
// Delay. The goroutine exists only between StartSpinner and Stop; outside a
// terminal no goroutine is started at all.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// clearLine returns the cursor to column one and erases the row.
const clearLine = "\r\x1b[2K"

// ASCII frames render in every console font, including Windows conhost.
const frames = `|/-\`

// SpinnerOptions tunes a spinner. Zero values select the defaults.
type SpinnerOptions struct {
	// Delay before the first frame. Default 300ms.
	Delay time.Duration
	// Interval between frames. Default 100ms.
	Interval time.Duration
}

// Spinner shows progress on a single terminal row until stopped.
type Spinner interface {
	// Update records progress in bytes; total <= 0 means unknown.
	// It is safe to call from any goroutine, also after Stop.
	Update(done, total int64)
	// Stop erases the spinner and returns the first write error.
	// It is idempotent and safe to call concurrently.
	Stop() error
}

type spinner struct {
	w        io.Writer
	label    string
	interval time.Duration
	done     atomic.Int64
	total    atomic.Int64
	stop     chan struct{}
	finished chan struct{}
	once     sync.Once
	// drawn and err are owned by the loop goroutine until finished is
	// closed, and by Stop afterwards.
	drawn bool
	err   error
}

type noopSpinner struct{}

func (noopSpinner) Update(int64, int64) {}
func (noopSpinner) Stop() error         { return nil }

// StartSpinner starts a spinner on w with the given label. When caps do not
// allow animation it returns a spinner that never writes.
func StartSpinner(w io.Writer, caps Caps, label string, opts SpinnerOptions) Spinner {
	if !caps.Animate {
		return noopSpinner{}
	}
	s := &spinner{
		w:        w,
		label:    label,
		interval: defaultDuration(opts.Interval, 100*time.Millisecond),
		stop:     make(chan struct{}),
		finished: make(chan struct{}),
	}
	go s.loop(defaultDuration(opts.Delay, 300*time.Millisecond))
	return s
}

func defaultDuration(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}

func (s *spinner) Update(done, total int64) {
	s.done.Store(done)
	s.total.Store(total)
}

func (s *spinner) Stop() error {
	s.once.Do(func() {
		close(s.stop)
		<-s.finished
		if s.drawn && s.err == nil {
			_, s.err = io.WriteString(s.w, clearLine)
		}
	})
	return s.err
}

func (s *spinner) loop(delay time.Duration) {
	defer close(s.finished)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-s.stop:
		return
	case <-timer.C:
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for frame := 0; s.draw(frame); frame++ {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
		}
	}
}

// draw writes one frame and reports whether drawing should continue.
func (s *spinner) draw(frame int) bool {
	line := clearLine + frames[frame%len(frames):frame%len(frames)+1] + " " +
		ProgressLine(s.label, s.done.Load(), s.total.Load())
	s.drawn = true
	_, s.err = io.WriteString(s.w, line)
	return s.err == nil
}
