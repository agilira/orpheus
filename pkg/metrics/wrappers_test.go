// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	newmetrics "github.com/agilira/orpheus/metrics"
	"github.com/agilira/orpheus/pkg/metrics"
)

// TestWrappersCallTheirTarget checks that every function of the facade
// returns what the function it wraps returns.
func TestWrappersCallTheirTarget(t *testing.T) {
	ctx := context.Background()

	var a, b bytes.Buffer
	metrics.NewJSONL(&a, metrics.Options{}).Counter("c_total", "help").Inc(ctx)
	newmetrics.NewJSONL(&b, newmetrics.Options{}).Counter("c_total", "help").Inc(ctx)
	stamp := regexp.MustCompile(`"time":"[^"]*"`)
	if a.Len() == 0 || stamp.ReplaceAllString(a.String(), "") != stamp.ReplaceAllString(b.String(), "") {
		t.Errorf("NewJSONL differs: %q vs %q", a.String(), b.String())
	}

	p1, p2 := metrics.NewPrometheus(metrics.Options{}), newmetrics.NewPrometheus(newmetrics.Options{})
	p1.Counter("c_total", "help").Inc(ctx)
	p2.Counter("c_total", "help").Inc(ctx)
	r1, r2 := httptest.NewRecorder(), httptest.NewRecorder()
	p1.ServeHTTP(r1, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	p2.ServeHTTP(r2, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if r1.Body.String() != r2.Body.String() {
		t.Errorf("NewPrometheus differs: %q vs %q", r1.Body.String(), r2.Body.String())
	}

	if _, err := metrics.Listen("no-port"); err == nil {
		t.Error("Listen accepted an address without a port")
	}
	ln, err := metrics.Listen(":0")
	if err != nil {
		t.Fatal(err)
	}
	if host, _, _ := net.SplitHostPort(ln.Addr().String()); !net.ParseIP(host).IsLoopback() {
		t.Errorf("Listen(\":0\") is not on loopback: %s", ln.Addr())
	}
	cctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- metrics.Serve(cctx, ln, p1) }()
	cancel()
	if err := <-done; err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Serve: %v", err)
	}
}
