// Package metrics keeps the reference MetricsCollector implementations at
// their original import path. The same API is available at
// github.com/agilira/orpheus/metrics; the names here are aliases of its
// types and wrappers of its functions.
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

	"github.com/agilira/orpheus/metrics"
	"github.com/agilira/orpheus/observe"
)

// Types, see package github.com/agilira/orpheus/metrics.
type (
	DropError  = metrics.DropError
	DropReason = metrics.DropReason
	Options    = metrics.Options
	Prometheus = metrics.Prometheus
)

// Drop reasons, see package github.com/agilira/orpheus/metrics.
const (
	ReasonInvalidName   = metrics.ReasonInvalidName
	ReasonInvalidValue  = metrics.ReasonInvalidValue
	ReasonInvalidBucket = metrics.ReasonInvalidBucket
	ReasonLabelCount    = metrics.ReasonLabelCount
	ReasonValueTooLong  = metrics.ReasonValueTooLong
	ReasonConflict      = metrics.ReasonConflict
	ReasonCardinality   = metrics.ReasonCardinality
	ReasonWriteFailed   = metrics.ReasonWriteFailed
)

// Listen calls metrics.Listen.
func Listen(addr string) (net.Listener, error) { return metrics.Listen(addr) }

// NewJSONL calls metrics.NewJSONL.
func NewJSONL(w io.Writer, opts Options) observe.MetricsCollector { return metrics.NewJSONL(w, opts) }

// NewPrometheus calls metrics.NewPrometheus.
func NewPrometheus(opts Options) Prometheus { return metrics.NewPrometheus(opts) }

// Serve calls metrics.Serve.
func Serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	return metrics.Serve(ctx, ln, h)
}
