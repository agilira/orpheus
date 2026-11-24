// registry.go: metric families, series and the bounds that protect them.
//
// WHY: every limit is checked under the same lock that creates the series,
// so concurrent callers cannot race past a bound. User callbacks (OnError)
// run only after the lock is released, so they may use the collector.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"sync"
)

type kind uint8

const (
	counterKind kind = iota
	gaugeKind
	histogramKind
)

var kindNames = [...]string{counterKind: "counter", gaugeKind: "gauge", histogramKind: "histogram"}

type op uint8

const (
	opCounterAdd op = iota
	opGaugeAdd
	opGaugeSet
	opObserve
)

var defaultBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type family struct {
	name    string
	help    string
	kind    kind
	labels  []string
	buckets []float64
	series  map[string]*series
}

type series struct {
	values []string
	value  float64  // counter or gauge value; sum for a histogram
	counts []uint64 // histogram only: per bucket, last one is +Inf
	count  uint64
}

type registry struct {
	opts     Options
	mu       sync.Mutex
	families map[string]*family
	taken    map[string]bool // every name that appears in the exposition
	dropped  map[DropReason]uint64
	nseries  int
	nbytes   int
}

// register returns the family, or nil when the registration is dropped.
func (r *registry) register(name, help string, k kind, buckets []float64, labels []string) *family {
	f, d := r.lockedRegister(name, help, k, buckets, labels)
	if d != nil {
		r.report(d)
	}
	return f
}

func (r *registry) lockedRegister(name, help string, k kind, buckets []float64, labels []string) (*family, *DropError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	buckets, d := validateRegistration(name, k, buckets, labels)
	if d == nil {
		var f *family
		if f, d = r.lookupOrCreate(name, help, k, buckets, labels); d == nil {
			return f, nil
		}
	}
	r.dropped[d.Reason]++
	return nil, d
}

func validateRegistration(name string, k kind, buckets []float64, labels []string) ([]float64, *DropError) {
	if !validMetricName(name) {
		return nil, &DropError{Metric: name, Reason: ReasonInvalidName, Detail: "invalid metric name"}
	}
	if err := checkLabelNames(labels, k == histogramKind); err != nil {
		return nil, &DropError{Metric: name, Reason: ReasonInvalidName, Detail: err.Error()}
	}
	if k != histogramKind {
		return nil, nil
	}
	b, err := checkBuckets(buckets)
	if err != nil {
		return nil, &DropError{Metric: name, Reason: ReasonInvalidBucket, Detail: err.Error()}
	}
	return b, nil
}

// checkBuckets returns a private copy of valid bounds; nil selects the
// defaults. +Inf is implicit and may not be listed.
func checkBuckets(b []float64) ([]float64, error) {
	if b == nil {
		return slices.Clone(defaultBuckets), nil
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("no buckets")
	}
	for i, v := range b {
		if math.IsNaN(v) || math.IsInf(v, 0) || (i > 0 && v <= b[i-1]) {
			return nil, fmt.Errorf("bucket bounds must be finite and strictly increasing")
		}
	}
	return slices.Clone(b), nil
}

func (r *registry) lookupOrCreate(name, help string, k kind, buckets []float64, labels []string) (*family, *DropError) {
	if f, ok := r.families[name]; ok {
		if f.kind == k && slices.Equal(f.labels, labels) && slices.Equal(f.buckets, buckets) {
			return f, nil
		}
		return nil, &DropError{Metric: name, Reason: ReasonConflict, Detail: "registered with a different type, labels or buckets"}
	}
	names := exposedNames(name, k)
	for _, n := range names {
		if r.taken[n] {
			return nil, &DropError{Metric: name, Reason: ReasonConflict, Detail: "name " + n + " already exposed"}
		}
	}
	if len(r.families) >= r.opts.MaxMetrics {
		return nil, &DropError{Metric: name, Reason: ReasonCardinality, Detail: "too many metrics"}
	}
	f := &family{name: name, help: help, kind: k, labels: slices.Clone(labels), buckets: buckets,
		series: make(map[string]*series)}
	r.families[name] = f
	for _, n := range names {
		r.taken[n] = true
	}
	return f, nil
}

// exposedNames lists the names a family writes, so that a histogram and a
// gauge cannot both produce "x_count".
func exposedNames(name string, k kind) []string {
	if k != histogramKind {
		return []string{name}
	}
	return []string{name, name + "_bucket", name + "_sum", name + "_count"}
}

// update applies one sample. f is nil for a dropped registration.
func (r *registry) update(f *family, o op, v float64, values []string) {
	if f == nil {
		return
	}
	if d := r.lockedUpdate(f, o, v, values); d != nil {
		r.report(d)
	}
}

func (r *registry) lockedUpdate(f *family, o op, v float64, values []string) *DropError {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.apply(f, o, v, values)
	if d != nil {
		d.Metric = f.name
		r.dropped[d.Reason]++
	}
	return d
}

func (r *registry) apply(f *family, o op, v float64, values []string) *DropError {
	if !validValue(o, v) {
		return &DropError{Reason: ReasonInvalidValue, Detail: fmt.Sprintf("value %v", v)}
	}
	s, d := r.seriesFor(f, values)
	if d != nil {
		return d
	}
	switch o {
	case opGaugeSet:
		s.value = v
	case opObserve:
		s.counts[sort.SearchFloat64s(f.buckets, v)]++
		s.count++
		s.value += v
	default:
		s.value += v
	}
	return nil
}

// validValue rejects NaN everywhere, and negative or infinite counter
// increments, which would make a counter decrease or stick.
func validValue(o op, v float64) bool {
	if math.IsNaN(v) {
		return false
	}
	return o != opCounterAdd || (v >= 0 && !math.IsInf(v, 1))
}

func (r *registry) seriesFor(f *family, values []string) (*series, *DropError) {
	if d := checkLabelValues(values, len(f.labels)); d != nil {
		return nil, d
	}
	key := seriesKey(values)
	if s, ok := f.series[key]; ok {
		return s, nil
	}
	cost := seriesCost(f, values)
	if r.nseries >= r.opts.MaxSeries || r.nbytes+cost > r.opts.MaxBytes {
		return nil, &DropError{Reason: ReasonCardinality, Detail: "series limit reached"}
	}
	s := &series{values: slices.Clone(values)}
	if f.kind == histogramKind {
		s.counts = make([]uint64, len(f.buckets)+1)
	}
	f.series[key] = s
	r.nseries++
	r.nbytes += cost
	return s, nil
}

// seriesKey is length-prefixed so that no choice of values can make two
// different label sets share a key.
func seriesKey(values []string) string {
	key := make([]byte, 0, 64)
	for _, v := range values {
		key = fmt.Appendf(key, "%d:%s", len(v), v)
	}
	return string(key)
}

// seriesCost is an upper bound of the bytes a series adds to the
// exposition: escaping at most doubles a value, a histogram repeats its
// labels on every bucket line plus _sum and _count.
func seriesCost(f *family, values []string) int {
	line := len(f.name) + 48
	for i, v := range values {
		line += len(f.labels[i]) + 2*len(v) + 4
	}
	if f.kind == histogramKind {
		return line * (len(f.buckets) + 3)
	}
	return line
}

// report passes a drop to OnError. Callers must not hold r.mu.
func (r *registry) report(d *DropError) {
	if r.opts.OnError != nil {
		r.opts.OnError(d)
	}
}

// fail counts and reports a drop that happened outside a registry update.
func (r *registry) fail(d *DropError) {
	r.mu.Lock()
	r.dropped[d.Reason]++
	r.mu.Unlock()
	r.report(d)
}
