// jsonl.go: an orpheus.MetricsCollector that writes one JSON object per
// sample to an io.Writer.
//
// WHY: not everyone runs Prometheus. A JSON Lines stream is a history that
// jq, Loki or a SIEM can read with no server. The collector keeps no series
// in memory: each sample is validated, encoded and written in one Write, so
// lines from concurrent goroutines never interleave. encoding/json escapes
// newlines and replaces invalid UTF-8, which keeps every sample on one line.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/agilira/orpheus/pkg/orpheus"
)

var opNames = [...]string{opCounterAdd: "add", opGaugeAdd: "add", opGaugeSet: "set", opObserve: "observe"}

type jsonl struct {
	reg *registry // registrations, conflicts and drop counts only
	mu  sync.Mutex
	w   io.Writer
}

type jsonRecord struct {
	Time   string            `json:"time"`
	Metric string            `json:"metric"`
	Type   string            `json:"type"`
	Op     string            `json:"op"`
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

// NewJSONL returns a collector writing to w. Options.MaxSeries and
// MaxBytes do not apply: nothing is retained per series.
func NewJSONL(w io.Writer, opts Options) orpheus.MetricsCollector {
	return &jsonl{reg: newRegistry(opts), w: w}
}

func (j *jsonl) Counter(name, help string, labels ...string) orpheus.Counter {
	return counter{j, j.reg.register(name, help, counterKind, nil, labels)}
}

func (j *jsonl) Gauge(name, help string, labels ...string) orpheus.Gauge {
	return gauge{j, j.reg.register(name, help, gaugeKind, nil, labels)}
}

func (j *jsonl) Histogram(name, help string, buckets []float64, labels ...string) orpheus.Histogram {
	return histogram{j, j.reg.register(name, help, histogramKind, buckets, labels)}
}

func (j *jsonl) update(f *family, o op, v float64, values []string) {
	if f == nil {
		return
	}
	line, d := encodeSample(f, o, v, values)
	if d == nil {
		d = j.write(line)
	}
	if d != nil {
		d.Metric = f.name
		j.reg.fail(d)
	}
}

// encodeSample validates a sample and returns its line. JSON has no
// infinities, so they are refused here in addition to the common rules.
func encodeSample(f *family, o op, v float64, values []string) ([]byte, *DropError) {
	if !validValue(o, v) || math.IsInf(v, 0) {
		return nil, &DropError{Reason: ReasonInvalidValue, Detail: fmt.Sprintf("value %v", v)}
	}
	if d := checkLabelValues(values, len(f.labels)); d != nil {
		return nil, d
	}
	rec := jsonRecord{
		Time:   time.Now().UTC().Format(time.RFC3339Nano),
		Metric: f.name,
		Type:   kindNames[f.kind],
		Op:     opNames[o],
		Value:  v,
	}
	if len(values) > 0 {
		rec.Labels = make(map[string]string, len(values))
		for i, l := range f.labels {
			rec.Labels[l] = values[i]
		}
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return nil, &DropError{Reason: ReasonInvalidValue, Detail: err.Error()}
	}
	return append(line, '\n'), nil
}

func (j *jsonl) write(line []byte) *DropError {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, err := j.w.Write(line); err != nil {
		return &DropError{Reason: ReasonWriteFailed, Detail: err.Error()}
	}
	return nil
}
