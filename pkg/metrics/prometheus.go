// prometheus.go: an in-memory orpheus.MetricsCollector served in the
// Prometheus text format.
//
// WHY: Orpheus defines MetricsCollector but no implementation, and the
// official client library is a large dependency for what is a simple, stable
// text format. This collector uses only the standard library. Because the
// interface cannot return errors, every rejected registration or sample is
// dropped, counted in orpheus_metrics_dropped_total and passed to OnError;
// a rejected registration returns a metric that does nothing.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/agilira/orpheus/pkg/orpheus"
)

const (
	contentType = "text/plain; version=0.0.4; charset=utf-8"
	droppedName = "orpheus_metrics_dropped_total"
	droppedHelp = "Metric registrations and samples dropped by the collector."
)

// Default bounds. They are generous for a CLI and still keep a flood of
// distinct label values from exhausting memory or the scraper.
const (
	defaultMaxMetrics = 1000
	defaultMaxSeries  = 10000
	defaultMaxBytes   = 16 << 20
)

// Options configures a collector. Zero values select the defaults.
type Options struct {
	// MaxMetrics bounds the number of registered metric families.
	MaxMetrics int
	// MaxSeries bounds the number of series across all families.
	MaxSeries int
	// MaxBytes bounds the estimated size of the exposition.
	MaxBytes int
	// OnError receives a *DropError for every dropped registration or
	// sample. It is called without internal locks held, so it may use the
	// collector. It must be safe for concurrent use.
	OnError func(error)
}

// Prometheus is a MetricsCollector that serves its metrics over HTTP.
type Prometheus interface {
	orpheus.MetricsCollector
	http.Handler
}

// NewPrometheus returns an empty collector.
func NewPrometheus(opts Options) Prometheus {
	return newRegistry(opts)
}

func newRegistry(opts Options) *registry {
	opts.MaxMetrics = positiveOr(opts.MaxMetrics, defaultMaxMetrics)
	opts.MaxSeries = positiveOr(opts.MaxSeries, defaultMaxSeries)
	opts.MaxBytes = positiveOr(opts.MaxBytes, defaultMaxBytes)
	return &registry{
		opts:     opts,
		families: make(map[string]*family),
		taken:    map[string]bool{droppedName: true},
		dropped:  make(map[DropReason]uint64),
	}
}

func positiveOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func (r *registry) Counter(name, help string, labels ...string) orpheus.Counter {
	return counter{r, r.register(name, help, counterKind, nil, labels)}
}

func (r *registry) Gauge(name, help string, labels ...string) orpheus.Gauge {
	return gauge{r, r.register(name, help, gaugeKind, nil, labels)}
}

func (r *registry) Histogram(name, help string, buckets []float64, labels ...string) orpheus.Histogram {
	return histogram{r, r.register(name, help, histogramKind, buckets, labels)}
}

// ServeHTTP answers GET and HEAD with the exposition. The request is not
// otherwise read: no query, header or body influences the response.
func (r *registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body := r.render()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if req.Method == http.MethodHead {
		return
	}
	if _, err := io.WriteString(w, body); err != nil {
		r.fail(&DropError{Metric: droppedName, Reason: ReasonWriteFailed, Detail: err.Error()})
	}
}

// Metric handles, shared by every collector in this package. A nil family
// means the registration was dropped; every method is then a no-op.

// sink receives the samples of a handle.
type sink interface {
	update(f *family, o op, v float64, values []string)
}

type counter struct {
	s sink
	f *family
}

func (c counter) Inc(_ context.Context, labels ...string) { c.s.update(c.f, opCounterAdd, 1, labels) }

func (c counter) Add(_ context.Context, v float64, labels ...string) {
	c.s.update(c.f, opCounterAdd, v, labels)
}

type gauge struct {
	s sink
	f *family
}

func (g gauge) Set(_ context.Context, v float64, labels ...string) {
	g.s.update(g.f, opGaugeSet, v, labels)
}

func (g gauge) Inc(_ context.Context, labels ...string) { g.s.update(g.f, opGaugeAdd, 1, labels) }

func (g gauge) Dec(_ context.Context, labels ...string) { g.s.update(g.f, opGaugeAdd, -1, labels) }

func (g gauge) Add(_ context.Context, v float64, labels ...string) {
	g.s.update(g.f, opGaugeAdd, v, labels)
}

type histogram struct {
	s sink
	f *family
}

func (h histogram) Observe(_ context.Context, v float64, labels ...string) {
	h.s.update(h.f, opObserve, v, labels)
}
