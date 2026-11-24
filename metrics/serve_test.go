// serve_test.go: tests for the metrics HTTP server.
//
// WHY: the endpoint exposes names chosen by the application (for Cerberus,
// the paths it protects). It must not reach the network unless asked to,
// must not be readable by a web page through DNS rebinding, must serve
// nothing but /metrics, and must not let slow clients hold it open.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListenDefaultsToLoopback(t *testing.T) {
	ln, err := Listen(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, ln)
	if ip := ln.Addr().(*net.TCPAddr).IP; !ip.IsLoopback() {
		t.Fatalf("empty host bound %v, want loopback", ip)
	}
}

func TestListenHonorsExplicitHost(t *testing.T) {
	ln, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeListener(t, ln)
	if !strings.HasPrefix(ln.Addr().String(), "127.0.0.1:") {
		t.Fatalf("bound %v", ln.Addr())
	}
}

func TestListenRejectsMalformedAddress(t *testing.T) {
	for _, addr := range []string{"", "9464", "host:port:extra", "localhost:99999"} {
		if ln, err := Listen(addr); err == nil {
			closeListener(t, ln)
			t.Errorf("Listen(%q) succeeded", addr)
		}
	}
}

func closeListener(t *testing.T, ln net.Listener) {
	t.Helper()
	if err := ln.Close(); err != nil {
		t.Error(err)
	}
}

func startServer(t *testing.T, addr string) (string, context.CancelFunc, chan error) {
	t.Helper()
	ln, err := Listen(addr)
	if err != nil {
		t.Fatal(err)
	}
	p := NewPrometheus(Options{})
	p.Gauge("up", "").Set(context.Background(), 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, ln, p) }()
	return ln.Addr().String(), cancel, done
}

func get(t *testing.T, addr, path, host string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

func TestServeOnlyMetricsPath(t *testing.T) {
	addr, cancel, _ := startServer(t, "127.0.0.1:0")
	defer cancel()
	if code, body := get(t, addr, "/metrics", ""); code != http.StatusOK || !strings.Contains(body, "\nup 1\n") {
		t.Fatalf("/metrics: %d %q", code, body)
	}
	for _, p := range []string{"/", "/metrics/", "/debug/pprof/", "/metricsx"} {
		if code, _ := get(t, addr, p, ""); code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", p, code)
		}
	}
}

func TestServeRejectsForeignHostOnLoopback(t *testing.T) {
	addr, cancel, _ := startServer(t, "127.0.0.1:0")
	defer cancel()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"evil.example", "evil.example:" + port, "127.0.0.1.evil.example", "localhost.evil:" + port} {
		if code, body := get(t, addr, "/metrics", h); code != http.StatusMisdirectedRequest || strings.Contains(body, "up") {
			t.Errorf("Host %q: status %d, body %q", h, code, body)
		}
	}
	for _, h := range []string{"localhost:" + port, "127.0.0.1:" + port, "[::1]:" + port, "LOCALHOST", "127.0.0.1"} {
		if code, _ := get(t, addr, "/metrics", h); code != http.StatusOK {
			t.Errorf("Host %q: status %d, want 200", h, code)
		}
	}
}

func TestServeAcceptsAnyHostWhenPublic(t *testing.T) {
	addr, cancel, _ := startServer(t, "0.0.0.0:0")
	defer cancel()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := get(t, "127.0.0.1:"+port, "/metrics", "metrics.internal"); code != http.StatusOK {
		t.Fatalf("public bind refused a named host: %d", code)
	}
}

func TestServeStopsWithContext(t *testing.T) {
	_, cancel, done := startServer(t, "127.0.0.1:0")
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v after cancel", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop")
	}
}

func TestServerHasTimeoutsAndHeaderLimit(t *testing.T) {
	s := newServer(http.NotFoundHandler(), false)
	if s.ReadHeaderTimeout <= 0 || s.ReadTimeout <= 0 || s.WriteTimeout <= 0 || s.IdleTimeout <= 0 {
		t.Fatalf("missing timeouts: %+v", s)
	}
	if s.MaxHeaderBytes <= 0 || s.MaxHeaderBytes > 16<<10 {
		t.Fatalf("MaxHeaderBytes = %d", s.MaxHeaderBytes)
	}
}

func FuzzLoopbackHost(f *testing.F) {
	for _, s := range []string{"localhost", "127.0.0.1:1", "[::1]", "evil", "127.0.0.1.evil", "[::1].evil", "localhost:", "%31%32%37.0.0.1"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, host string) {
		if !loopbackHost(host) {
			return
		}
		h := host
		if hh, _, err := net.SplitHostPort(host); err == nil {
			h = hh
		}
		h = strings.Trim(h, "[]")
		ip := net.ParseIP(h)
		if !strings.EqualFold(h, "localhost") && (ip == nil || !ip.IsLoopback()) {
			t.Fatalf("loopbackHost accepted %q", host)
		}
	})
}
