// prometheus_test.go: tests for the in-memory Prometheus collector.
//
// WHY: the collector implements orpheus.MetricsCollector, whose methods
// cannot return errors, and is fed values that often come from outside. The
// tests pin what happens to every bad input (dropped, counted, reported,
// never a panic and never a different series), the memory bound under a
// flood of label values, and the exact text a scraper receives.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/agilira/orpheus/pkg/metrics"
	"github.com/agilira/orpheus/pkg/orpheus"
)

var ctx = context.Background()

// Compile-time proof that the collector satisfies the Orpheus interface.
var _ orpheus.MetricsCollector = metrics.NewPrometheus(metrics.Options{})

type errorLog struct {
	mu   sync.Mutex
	errs []error
}

func (l *errorLog) record(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, err)
}

func (l *errorLog) reasons() []metrics.DropReason {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []metrics.DropReason
	for _, err := range l.errs {
		var d *metrics.DropError
		if errors.As(err, &d) {
			out = append(out, d.Reason)
		}
	}
	return out
}

func newCollector(t *testing.T, opts metrics.Options) (metrics.Prometheus, *errorLog) {
	t.Helper()
	log := &errorLog{}
	opts.OnError = log.record
	return metrics.NewPrometheus(opts), log
}

func scrape(t *testing.T, p metrics.Prometheus) string {
	t.Helper()
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("scrape status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("Content-Type %q", ct)
	}
	return rec.Body.String()
}

func TestCounterExposition(t *testing.T) {
	p, log := newCollector(t, metrics.Options{})
	c := p.Counter("cerberus_transitions_total", "Status transitions.", "probe", "to")
	c.Inc(ctx, "~/.claude.json", "drifted")
	c.Add(ctx, 2, "~/.claude.json", "drifted")
	c.Inc(ctx, "a", "ok")
	got := scrape(t, p)
	want := "# HELP cerberus_transitions_total Status transitions.\n" +
		"# TYPE cerberus_transitions_total counter\n" +
		"cerberus_transitions_total{probe=\"a\",to=\"ok\"} 1\n" +
		"cerberus_transitions_total{probe=\"~/.claude.json\",to=\"drifted\"} 3\n"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s\nwant prefix:\n%s", got, want)
	}
	if r := log.reasons(); len(r) != 0 {
		t.Fatalf("unexpected drops: %v", r)
	}
}

func TestGaugeOperations(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	g := p.Gauge("g", "A gauge.")
	g.Set(ctx, 5)
	g.Inc(ctx)
	g.Dec(ctx)
	g.Dec(ctx)
	g.Add(ctx, -0.5)
	if !strings.Contains(scrape(t, p), "\ng 3.5\n") {
		t.Fatalf("gauge value wrong:\n%s", scrape(t, p))
	}
}

func TestHistogramExposition(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	h := p.Histogram("d_seconds", "Durations.", []float64{0.25, 1}, "op")
	for _, v := range []float64{0.125, 0.25, 0.5, 2} {
		h.Observe(ctx, v, "verify")
	}
	want := "# HELP d_seconds Durations.\n" +
		"# TYPE d_seconds histogram\n" +
		"d_seconds_bucket{op=\"verify\",le=\"0.25\"} 2\n" +
		"d_seconds_bucket{op=\"verify\",le=\"1\"} 3\n" +
		"d_seconds_bucket{op=\"verify\",le=\"+Inf\"} 4\n" +
		"d_seconds_sum{op=\"verify\"} 2.875\n" +
		"d_seconds_count{op=\"verify\"} 4\n"
	if got := scrape(t, p); !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s\nwant prefix:\n%s", got, want)
	}
}

func TestHistogramDefaultBuckets(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	p.Histogram("h", "", nil).Observe(ctx, 0.3)
	if got := scrape(t, p); strings.Count(got, "h_bucket{") != 12 {
		t.Fatalf("want 11 default buckets plus +Inf:\n%s", got)
	}
}

func TestSameRegistrationReturnsSameMetric(t *testing.T) {
	p, log := newCollector(t, metrics.Options{})
	p.Counter("c", "", "a").Inc(ctx, "x")
	p.Counter("c", "", "a").Inc(ctx, "x")
	if !strings.Contains(scrape(t, p), "c{a=\"x\"} 2\n") {
		t.Fatal("re-registration did not return the same counter")
	}
	if len(log.reasons()) != 0 {
		t.Fatalf("unexpected drops: %v", log.reasons())
	}
}

func TestRegistrationsThatAreDropped(t *testing.T) {
	cases := []struct {
		name     string
		register func(p metrics.Prometheus)
		reason   metrics.DropReason
	}{
		{"invalid metric name", func(p metrics.Prometheus) { p.Counter("a b", "") }, metrics.ReasonInvalidName},
		{"invalid label name", func(p metrics.Prometheus) { p.Gauge("g", "", "a-b") }, metrics.ReasonInvalidName},
		{"le on histogram", func(p metrics.Prometheus) { p.Histogram("h", "", nil, "le") }, metrics.ReasonInvalidName},
		{"kind conflict", func(p metrics.Prometheus) { p.Counter("x", ""); p.Gauge("x", "") }, metrics.ReasonConflict},
		{"label conflict", func(p metrics.Prometheus) { p.Counter("x", "", "a"); p.Counter("x", "", "b") }, metrics.ReasonConflict},
		{"bucket conflict", func(p metrics.Prometheus) {
			p.Histogram("x", "", []float64{1})
			p.Histogram("x", "", []float64{2})
		}, metrics.ReasonConflict},
		{"histogram suffix taken", func(p metrics.Prometheus) { p.Gauge("x_count", ""); p.Histogram("x", "", nil) }, metrics.ReasonConflict},
		{"name taken by histogram", func(p metrics.Prometheus) { p.Histogram("x", "", nil); p.Gauge("x_sum", "") }, metrics.ReasonConflict},
		{"reserved name", func(p metrics.Prometheus) { p.Counter("orpheus_metrics_dropped_total", "") }, metrics.ReasonConflict},
		{"unsorted buckets", func(p metrics.Prometheus) { p.Histogram("h", "", []float64{2, 1}) }, metrics.ReasonInvalidBucket},
		{"duplicate buckets", func(p metrics.Prometheus) { p.Histogram("h", "", []float64{1, 1}) }, metrics.ReasonInvalidBucket},
		{"NaN bucket", func(p metrics.Prometheus) { p.Histogram("h", "", []float64{math.NaN()}) }, metrics.ReasonInvalidBucket},
		{"Inf bucket", func(p metrics.Prometheus) { p.Histogram("h", "", []float64{math.Inf(1)}) }, metrics.ReasonInvalidBucket},
		{"empty buckets", func(p metrics.Prometheus) { p.Histogram("h", "", []float64{}) }, metrics.ReasonInvalidBucket},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, log := newCollector(t, metrics.Options{})
			tc.register(p)
			r := log.reasons()
			if len(r) != 1 || r[0] != tc.reason {
				t.Fatalf("reasons = %v, want [%s]", r, tc.reason)
			}
			if !strings.Contains(scrape(t, p), `orpheus_metrics_dropped_total{reason="`+string(tc.reason)+`"} 1`) {
				t.Fatalf("drop not exposed:\n%s", scrape(t, p))
			}
		})
	}
}

func TestDroppedMetricIsUsableNoOp(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	p.Counter("bad name", "").Inc(ctx)
	p.Gauge("bad name", "").Set(ctx, 1)
	p.Histogram("bad name", "", nil).Observe(ctx, 1)
	if strings.Contains(scrape(t, p), "bad") {
		t.Fatal("a dropped metric was exposed")
	}
}

func TestSamplesThatAreDropped(t *testing.T) {
	cases := []struct {
		name   string
		sample func(p metrics.Prometheus)
		reason metrics.DropReason
	}{
		{"too few label values", func(p metrics.Prometheus) { p.Counter("c", "", "a", "b").Inc(ctx, "x") }, metrics.ReasonLabelCount},
		{"too many label values", func(p metrics.Prometheus) { p.Gauge("g", "").Set(ctx, 1, "x") }, metrics.ReasonLabelCount},
		{"label value too long", func(p metrics.Prometheus) {
			p.Counter("c", "", "a").Inc(ctx, strings.Repeat("x", 1025))
		}, metrics.ReasonValueTooLong},
		{"negative counter add", func(p metrics.Prometheus) { p.Counter("c", "").Add(ctx, -1) }, metrics.ReasonInvalidValue},
		{"NaN counter add", func(p metrics.Prometheus) { p.Counter("c", "").Add(ctx, math.NaN()) }, metrics.ReasonInvalidValue},
		{"Inf counter add", func(p metrics.Prometheus) { p.Counter("c", "").Add(ctx, math.Inf(1)) }, metrics.ReasonInvalidValue},
		{"NaN gauge set", func(p metrics.Prometheus) { p.Gauge("g", "").Set(ctx, math.NaN()) }, metrics.ReasonInvalidValue},
		{"NaN gauge add", func(p metrics.Prometheus) { p.Gauge("g", "").Add(ctx, math.NaN()) }, metrics.ReasonInvalidValue},
		{"NaN observation", func(p metrics.Prometheus) { p.Histogram("h", "", nil).Observe(ctx, math.NaN()) }, metrics.ReasonInvalidValue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, log := newCollector(t, metrics.Options{})
			tc.sample(p)
			if r := log.reasons(); len(r) != 1 || r[0] != tc.reason {
				t.Fatalf("reasons = %v, want [%s]", r, tc.reason)
			}
		})
	}
}

func TestCardinalityIsBounded(t *testing.T) {
	p, log := newCollector(t, metrics.Options{MaxSeries: 3})
	c := p.Counter("c", "", "path")
	for i := range 100 {
		c.Inc(ctx, "/tmp/"+strconv.Itoa(i))
	}
	c.Inc(ctx, "/tmp/0")
	out := scrape(t, p)
	if n := strings.Count(out, "\nc{"); n != 3 {
		t.Fatalf("%d series kept, want 3:\n%s", n, out)
	}
	if !strings.Contains(out, "c{path=\"/tmp/0\"} 2\n") {
		t.Fatalf("an existing series stopped counting:\n%s", out)
	}
	if !strings.Contains(out, `orpheus_metrics_dropped_total{reason="cardinality"} 97`) {
		t.Fatalf("cardinality drops not exposed:\n%s", out)
	}
	if len(log.reasons()) != 97 {
		t.Fatalf("%d reports, want 97", len(log.reasons()))
	}
}

func TestExpositionSizeIsBounded(t *testing.T) {
	p, log := newCollector(t, metrics.Options{MaxBytes: 64 << 10})
	c := p.Counter("c", "", "path")
	for i := range 1000 {
		c.Inc(ctx, strconv.Itoa(i)+strings.Repeat("x", 1000))
	}
	if size := len(scrape(t, p)); size > 64<<10+1024 {
		t.Fatalf("exposition is %d bytes, budget 64 KiB", size)
	}
	if r := log.reasons(); len(r) == 0 || r[0] != metrics.ReasonCardinality {
		t.Fatalf("budget overflow not reported: %v", r)
	}
}

func TestMetricCountIsBounded(t *testing.T) {
	p, log := newCollector(t, metrics.Options{MaxMetrics: 2})
	for i := range 5 {
		p.Gauge("g"+strconv.Itoa(i), "").Set(ctx, 1)
	}
	if r := log.reasons(); len(r) != 3 || r[0] != metrics.ReasonCardinality {
		t.Fatalf("reasons = %v, want 3 cardinality drops", r)
	}
}

func TestOnErrorMayUseTheCollector(t *testing.T) {
	var p metrics.Prometheus
	p = metrics.NewPrometheus(metrics.Options{OnError: func(error) {
		p.Counter("errors_total", "").Inc(ctx)
	}})
	p.Counter("bad name", "")
	if !strings.Contains(scrape(t, p), "errors_total 1\n") {
		t.Fatal("OnError could not use the collector")
	}
}

func TestServeHTTPMethods(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest(m, "/metrics", strings.NewReader("x")))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status %d, want 405", m, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/metrics", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Errorf("HEAD: status %d, body %d bytes", rec.Code, rec.Body.Len())
	}
}

func TestHostileLabelValueCannotForgeSeries(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	p.Gauge("cerberus_probe_status", "", "probe").Set(ctx, 0, "x\"} 0\ncerberus_probe_status{probe=\"evil\"} 1\n#")
	out := scrape(t, p)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.HasPrefix(line, `cerberus_probe_status{probe="evil"}`) {
			t.Fatalf("label value forged a series:\n%s", out)
		}
	}
}

func TestConcurrentUse(t *testing.T) {
	p, _ := newCollector(t, metrics.Options{})
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 200 {
				p.Counter("c", "", "w").Inc(ctx, strconv.Itoa(i))
				p.Gauge("g", "").Set(ctx, float64(j))
				p.Histogram("h", "", nil).Observe(ctx, float64(j))
				if j%50 == 0 {
					scrape(t, p)
				}
			}
		}()
	}
	wg.Wait()
	if !strings.Contains(scrape(t, p), "h_count 1600\n") {
		t.Fatal("observations lost under concurrency")
	}
}

// The grammar of the exposition format, strict enough that a forged line
// or an unescaped quote cannot match.
var (
	labelPair   = `[a-zA-Z_][a-zA-Z0-9_]*="(?:[^"\\\n]|\\[\\"n])*"`
	sampleLine  = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*(\{` + labelPair + `(,` + labelPair + `)*\})? \S+$`)
	commentLine = regexp.MustCompile(`^# (HELP|TYPE) [a-zA-Z_:][a-zA-Z0-9_:]* [^\n]*$`)
)

func FuzzCollector(f *testing.F) {
	f.Add("c", "l", "v", 1.0)
	f.Add("a b", "__x", "x\"}\n", math.NaN())
	f.Fuzz(func(t *testing.T, name, label, value string, v float64) {
		p := metrics.NewPrometheus(metrics.Options{MaxSeries: 4})
		p.Counter(name, name, label).Add(ctx, v, value)
		p.Gauge(name+"_g", "", label).Set(ctx, v, value)
		p.Histogram(name+"_h", "", nil, label).Observe(ctx, v, value)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		for _, line := range strings.Split(strings.TrimSuffix(rec.Body.String(), "\n"), "\n") {
			if !commentLine.MatchString(line) && !sampleLine.MatchString(line) {
				t.Fatalf("malformed line %q in:\n%s", line, rec.Body.String())
			}
		}
	})
}

type brokenWriter struct{ *httptest.ResponseRecorder }

func (*brokenWriter) Write([]byte) (int, error) { return 0, errors.New("connection reset") }

// WriteString shadows the recorder's own, which io.WriteString prefers.
func (*brokenWriter) WriteString(string) (int, error) { return 0, errors.New("connection reset") }

func TestScrapeWriteFailureIsReported(t *testing.T) {
	p, log := newCollector(t, metrics.Options{})
	w := &brokenWriter{httptest.NewRecorder()}
	p.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if r := log.reasons(); len(r) != 1 || r[0] != metrics.ReasonWriteFailed {
		t.Fatalf("reasons = %v, want [write_failed]", r)
	}
}
