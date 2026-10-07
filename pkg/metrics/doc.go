// Package metrics provides reference implementations of
// orpheus.MetricsCollector that depend only on the standard library:
//
//   - NewPrometheus keeps metrics in memory and serves them in the
//     Prometheus text format; Listen and Serve expose it with safe defaults.
//   - NewJSONL writes one JSON object per sample to an io.Writer.
//
// MetricsCollector methods cannot return errors and their label values often
// come from outside the program. Both collectors therefore validate every
// registration and sample, drop what is invalid or over a bound, count drops
// in orpheus_metrics_dropped_total and report them to Options.OnError. They
// never panic and never rewrite a sample into a different series.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0
package metrics
