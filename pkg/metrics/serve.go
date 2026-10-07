// serve.go: serving a collector over HTTP with safe defaults.
//
// WHY: a metrics endpoint publishes names chosen by the application, which
// for a security tool are the paths it protects. Three defaults follow:
// an address without a host binds loopback, not every interface; on
// loopback, requests must name a loopback host, so a web page cannot read
// the endpoint through DNS rebinding; and only /metrics is served, with
// timeouts and a header limit so slow clients cannot hold the server.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	loopbackDefault = "127.0.0.1"
	shutdownTimeout = 5 * time.Second
)

// Listen opens a TCP listener on addr ("host:port"). An empty host means
// loopback; use "0.0.0.0:port" or "[::]:port" to listen on every interface.
func Listen(addr string) (net.Listener, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = loopbackDefault
	}
	return net.Listen("tcp", net.JoinHostPort(host, port))
}

// Serve serves h at /metrics on ln until ctx is done, then shuts down
// gracefully and returns nil. Any other error from the server is returned.
// When ln is bound to a loopback address, requests whose Host header does
// not name a loopback host are refused with 421.
func Serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := newServer(h, isLoopback(ln))
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newServer(h http.Handler, loopback bool) *http.Server {
	if loopback {
		h = hostGuard(h)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", h)
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
}

func isLoopback(ln net.Listener) bool {
	addr, ok := ln.Addr().(*net.TCPAddr)
	return ok && addr.IP.IsLoopback()
}

func hostGuard(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			http.Error(w, "misdirected request", http.StatusMisdirectedRequest)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// loopbackHost reports whether a Host header names this machine: localhost
// or a loopback IP literal, with or without a port. Any other name may
// resolve to 127.0.0.1 only because an attacker's DNS says so.
func loopbackHost(host string) bool {
	h := host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		h = hh
	}
	h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
