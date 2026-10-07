// jsonl_test.go: tests for the JSON Lines collector.
//
// WHY: a JSONL file is read line by line by other tools (jq, Loki, a SIEM).
// One sample must be exactly one line of valid JSON whatever the label
// values contain, lines from concurrent goroutines must never interleave,
// and a value JSON cannot represent must be dropped, not written broken.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics_test

import (
	"encoding/json"
	"errors"
	"maps"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agilira/orpheus/pkg/metrics"
	"github.com/agilira/orpheus/pkg/orpheus"
)

var _ orpheus.MetricsCollector = metrics.NewJSONL(&syncBuffer{}, metrics.Options{})

type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type record struct {
	Time   string            `json:"time"`
	Metric string            `json:"metric"`
	Type   string            `json:"type"`
	Op     string            `json:"op"`
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

func records(t *testing.T, out string) []record {
	t.Helper()
	var recs []record
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		var r record
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line is not JSON: %q: %v", line, err)
		}
		if _, err := time.Parse(time.RFC3339Nano, r.Time); err != nil {
			t.Fatalf("bad time %q: %v", r.Time, err)
		}
		recs = append(recs, r)
	}
	return recs
}

func TestJSONLWritesOneLinePerSample(t *testing.T) {
	var buf syncBuffer
	log := &errorLog{}
	j := metrics.NewJSONL(&buf, metrics.Options{OnError: log.record})
	j.Counter("c_total", "", "probe").Add(ctx, 2, "~/.claude.json")
	g := j.Gauge("g", "")
	g.Set(ctx, 5)
	g.Inc(ctx)
	g.Dec(ctx)
	j.Histogram("h", "", nil).Observe(ctx, 0.5)
	recs := records(t, buf.String())
	want := []record{
		{Metric: "c_total", Type: "counter", Op: "add", Labels: map[string]string{"probe": "~/.claude.json"}, Value: 2},
		{Metric: "g", Type: "gauge", Op: "set", Value: 5},
		{Metric: "g", Type: "gauge", Op: "add", Value: 1},
		{Metric: "g", Type: "gauge", Op: "add", Value: -1},
		{Metric: "h", Type: "histogram", Op: "observe", Value: 0.5},
	}
	if len(recs) != len(want) {
		t.Fatalf("%d records, want %d:\n%s", len(recs), len(want), buf.String())
	}
	for i, w := range want {
		if !sameRecord(recs[i], w) {
			t.Errorf("record %d = %+v, want %+v", i, recs[i], w)
		}
	}
	if len(log.reasons()) != 0 {
		t.Fatalf("unexpected drops: %v", log.reasons())
	}
}

// sameRecord compares everything but the time.
func sameRecord(a, b record) bool {
	return a.Metric == b.Metric && a.Type == b.Type && a.Op == b.Op &&
		a.Value == b.Value && maps.Equal(a.Labels, b.Labels)
}

func TestJSONLHostileLabelValueStaysInItsLine(t *testing.T) {
	var buf syncBuffer
	j := metrics.NewJSONL(&buf, metrics.Options{})
	v := "x\"}\n{\"metric\":\"admin\",\"value\":1}\r\u2028"
	j.Gauge("g", "", "probe").Set(ctx, 1, v)
	out := buf.String()
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("one sample produced %d lines: %q", strings.Count(out, "\n"), out)
	}
	if recs := records(t, out); recs[0].Labels["probe"] != v {
		t.Fatalf("label value changed: %q", recs[0].Labels["probe"])
	}
}

func TestJSONLDrops(t *testing.T) {
	cases := []struct {
		name   string
		sample func(c orpheus.MetricsCollector)
		reason metrics.DropReason
	}{
		{"invalid name", func(c orpheus.MetricsCollector) { c.Counter("a b", "").Inc(ctx) }, metrics.ReasonInvalidName},
		{"conflict", func(c orpheus.MetricsCollector) { c.Counter("x", ""); c.Gauge("x", "").Set(ctx, 1) }, metrics.ReasonConflict},
		{"label count", func(c orpheus.MetricsCollector) { c.Gauge("g", "", "a").Set(ctx, 1) }, metrics.ReasonLabelCount},
		{"NaN", func(c orpheus.MetricsCollector) { c.Gauge("g", "").Set(ctx, math.NaN()) }, metrics.ReasonInvalidValue},
		{"Inf gauge", func(c orpheus.MetricsCollector) { c.Gauge("g", "").Set(ctx, math.Inf(-1)) }, metrics.ReasonInvalidValue},
		{"Inf observation", func(c orpheus.MetricsCollector) { c.Histogram("h", "", nil).Observe(ctx, math.Inf(1)) }, metrics.ReasonInvalidValue},
		{"negative counter", func(c orpheus.MetricsCollector) { c.Counter("c", "").Add(ctx, -1) }, metrics.ReasonInvalidValue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf syncBuffer
			log := &errorLog{}
			tc.sample(metrics.NewJSONL(&buf, metrics.Options{OnError: log.record}))
			if r := log.reasons(); len(r) != 1 || r[0] != tc.reason {
				t.Fatalf("reasons = %v, want [%s]", r, tc.reason)
			}
			if buf.String() != "" {
				t.Fatalf("a dropped sample was written: %q", buf.String())
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestJSONLWriteFailureIsReported(t *testing.T) {
	log := &errorLog{}
	j := metrics.NewJSONL(failingWriter{}, metrics.Options{OnError: log.record})
	j.Counter("c", "").Inc(ctx)
	j.Counter("c", "").Inc(ctx)
	if r := log.reasons(); len(r) != 2 || r[0] != metrics.ReasonWriteFailed {
		t.Fatalf("reasons = %v, want two write_failed", r)
	}
}

func TestJSONLConcurrentLinesDoNotInterleave(t *testing.T) {
	var buf syncBuffer
	j := metrics.NewJSONL(&buf, metrics.Options{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := j.Counter("c", "", "v")
			for range 200 {
				c.Inc(ctx, strings.Repeat("x", 512))
			}
		}()
	}
	wg.Wait()
	if n := len(records(t, buf.String())); n != 1600 {
		t.Fatalf("%d records, want 1600", n)
	}
}

func FuzzJSONL(f *testing.F) {
	f.Add("g", "l", "x\n\"}", 1.0)
	f.Add("a b", "__", "\xff", math.Inf(1))
	f.Fuzz(func(t *testing.T, name, label, value string, v float64) {
		var buf syncBuffer
		j := metrics.NewJSONL(&buf, metrics.Options{})
		j.Gauge(name, "", label).Set(ctx, v, value)
		j.Counter(name+"_total", "", label).Add(ctx, v, value)
		out := buf.String()
		if strings.Count(out, "\n") > 2 {
			t.Fatalf("two samples produced %d lines: %q", strings.Count(out, "\n"), out)
		}
		records(t, out)
	})
}
