// facade_observe.go: the observe API under its v1.11.0 names
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import "github.com/agilira/orpheus/observe"

// Observability types, see package observe.
type (
	Logger           = observe.Logger
	AuditLogger      = observe.AuditLogger
	Tracer           = observe.Tracer
	Span             = observe.Span
	MetricsCollector = observe.MetricsCollector
	Counter          = observe.Counter
	Gauge            = observe.Gauge
	Histogram        = observe.Histogram
	Field            = observe.Field
	SpanOption       = observe.SpanOption
	ErrorOption      = observe.ErrorOption
	StatusCode       = observe.StatusCode
)

// Span status codes, see package observe.
const (
	StatusCodeUnset = observe.StatusCodeUnset
	StatusCodeOK    = observe.StatusCodeOK
	StatusCodeError = observe.StatusCodeError
)

// StringField calls observe.StringField.
func StringField(key, value string) Field { return observe.StringField(key, value) }

// IntField calls observe.IntField.
func IntField(key string, value int) Field { return observe.IntField(key, value) }

// Float64Field calls observe.Float64Field.
func Float64Field(key string, value float64) Field { return observe.Float64Field(key, value) }

// BoolField calls observe.BoolField.
func BoolField(key string, value bool) Field { return observe.BoolField(key, value) }

// ErrorField calls observe.ErrorField.
func ErrorField(err error) Field { return observe.ErrorField(err) }
