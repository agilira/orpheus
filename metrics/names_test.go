// names_test.go: tests for metric and label name validation.
//
// WHY: names end up verbatim in the exposition format, unquoted. A name that
// is not checked against the grammar can inject lines, fake series or break
// every scraper reading the endpoint.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"errors"
	"strings"
	"testing"
)

func TestValidMetricName(t *testing.T) {
	for _, n := range []string{"a", "_a", ":a", "cerberus_probe_status", "a:b_c9"} {
		if !validMetricName(n) {
			t.Errorf("validMetricName(%q) = false", n)
		}
	}
	bad := []string{
		"", "9a", "a-b", "a b", "a\nb", "a{b}", "é", "a\"", "a\x00",
		strings.Repeat("a", maxNameLength+1),
	}
	for _, n := range bad {
		if validMetricName(n) {
			t.Errorf("validMetricName(%q) = true", n)
		}
	}
}

func TestValidLabelName(t *testing.T) {
	for _, n := range []string{"a", "_a", "probe", "key_id2"} {
		if !validLabelName(n) {
			t.Errorf("validLabelName(%q) = false", n)
		}
	}
	for _, n := range []string{"", "9a", "a:b", "__name__", "__x", "a=b", "a\"", "le\n"} {
		if validLabelName(n) {
			t.Errorf("validLabelName(%q) = true", n)
		}
	}
}

func TestCheckLabelNames(t *testing.T) {
	cases := []struct {
		name   string
		labels []string
		hist   bool
		ok     bool
	}{
		{"none", nil, false, true},
		{"valid", []string{"probe", "kind"}, false, true},
		{"duplicate", []string{"probe", "probe"}, false, false},
		{"invalid", []string{"pro be"}, false, false},
		{"le on histogram", []string{"le"}, true, false},
		{"le on counter", []string{"le"}, false, true},
		{"too many", make([]string, maxLabels+1), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkLabelNames(tc.labels, tc.hist)
			if (err == nil) != tc.ok {
				t.Fatalf("checkLabelNames(%q) = %v, want ok=%v", tc.labels, err, tc.ok)
			}
		})
	}
}

func TestCheckLabelValues(t *testing.T) {
	if err := checkLabelValues([]string{"a", "b"}, 2); err != nil {
		t.Fatal(err)
	}
	var d *DropError
	if err := checkLabelValues([]string{"a"}, 2); !errors.As(err, &d) || d.Reason != ReasonLabelCount {
		t.Fatalf("wrong count: %v", err)
	}
	long := []string{strings.Repeat("x", maxLabelValueLength+1)}
	if err := checkLabelValues(long, 1); !errors.As(err, &d) || d.Reason != ReasonValueTooLong {
		t.Fatalf("long value: %v", err)
	}
}

func TestDropErrorMessageIsSanitized(t *testing.T) {
	err := &DropError{Metric: "a\nb", Reason: ReasonInvalidName, Detail: "x\x1b[2J"}
	if strings.ContainsAny(err.Error(), "\n\x1b") {
		t.Fatalf("DropError leaks raw controls: %q", err.Error())
	}
}

func FuzzNames(f *testing.F) {
	for _, s := range []string{"a", "a_b:c", "\n", "a{b=\"c\"}", "__name__", "é"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if validMetricName(s) && strings.ContainsAny(s, "{}\"\\ \n\r\t=,#") {
			t.Fatalf("validMetricName accepted %q", s)
		}
		if validLabelName(s) && strings.ContainsAny(s, "{}\"\\ \n\r\t=,#:") {
			t.Fatalf("validLabelName accepted %q", s)
		}
	})
}
