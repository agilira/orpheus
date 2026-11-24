// names.go: validation of metric names, label names and label values.
//
// WHY: the MetricsCollector interface cannot return errors, and the values
// passed to it often come from outside (a file path, a server name). Every
// sample is therefore checked here and either accepted as is or dropped with
// a reason; nothing is silently rewritten into a different series.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"fmt"
	"strconv"
)

// Bounds on what a caller may register. They keep a single hostile value
// from turning a metrics endpoint into a memory or bandwidth sink.
const (
	maxNameLength       = 200
	maxLabels           = 16
	maxLabelValueLength = 1024
)

// DropReason says why a registration or a sample was dropped.
type DropReason string

// Drop reasons, also used as the "reason" label of the dropped counter.
const (
	ReasonInvalidName   DropReason = "invalid_name"
	ReasonConflict      DropReason = "conflict"
	ReasonLabelCount    DropReason = "label_count"
	ReasonValueTooLong  DropReason = "value_too_long"
	ReasonInvalidValue  DropReason = "invalid_value"
	ReasonCardinality   DropReason = "cardinality"
	ReasonWriteFailed   DropReason = "write_failed"
	ReasonInvalidBucket DropReason = "invalid_buckets"
)

// DropError describes a dropped registration or sample. It is passed to
// Options.OnError.
type DropError struct {
	Metric string
	Reason DropReason
	Detail string
}

// Error quotes the metric name and the detail, since both may contain
// caller-supplied text that would otherwise reach a log or a terminal raw.
func (e *DropError) Error() string {
	return fmt.Sprintf("metrics: %s dropped (%s): %s",
		strconv.Quote(e.Metric), e.Reason, strconv.Quote(e.Detail))
}

func validMetricName(s string) bool {
	if s == "" || len(s) > maxNameLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isNameChar(c, i == 0) && c != ':' {
			return false
		}
	}
	return true
}

// validLabelName also rejects the "__" prefix, which Prometheus reserves.
func validLabelName(s string) bool {
	if s == "" || len(s) > maxNameLength || (len(s) >= 2 && s[:2] == "__") {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isNameChar(s[i], i == 0) {
			return false
		}
	}
	return true
}

func isNameChar(c byte, first bool) bool {
	letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	return letter || (!first && c >= '0' && c <= '9')
}

// checkLabelNames validates the label names of a registration. Histograms
// may not use "le", which carries their bucket bounds.
func checkLabelNames(labels []string, histogram bool) error {
	if len(labels) > maxLabels {
		return fmt.Errorf("%d labels, at most %d allowed", len(labels), maxLabels)
	}
	seen := make(map[string]bool, len(labels))
	for _, l := range labels {
		if !validLabelName(l) || seen[l] || (histogram && l == "le") {
			return fmt.Errorf("invalid or duplicate label name %q", l)
		}
		seen[l] = true
	}
	return nil
}

// checkLabelValues validates the values passed with a sample.
func checkLabelValues(values []string, want int) *DropError {
	if len(values) != want {
		return &DropError{Reason: ReasonLabelCount,
			Detail: fmt.Sprintf("%d label values, want %d", len(values), want)}
	}
	for _, v := range values {
		if len(v) > maxLabelValueLength {
			return &DropError{Reason: ReasonValueTooLong,
				Detail: fmt.Sprintf("label value of %d bytes, at most %d", len(v), maxLabelValueLength)}
		}
	}
	return nil
}
