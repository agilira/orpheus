// expose.go: primitives of the Prometheus text exposition format 0.0.4.
//
// WHY: the format is line based and only label values and help text carry
// caller-supplied strings. Escaping them exactly as the specification says,
// and forcing valid UTF-8, is what keeps one value from forging lines.
// Names are never escaped here: they are validated at registration.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"math"
	"strconv"
	"strings"
)

var (
	labelValueEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	helpEscaper       = strings.NewReplacer(`\`, `\\`, "\n", `\n`)
)

// escapeLabelValue escapes a value for use between double quotes. Invalid
// UTF-8 is replaced, because the format requires UTF-8 and scrapers differ
// in how they reject it.
func escapeLabelValue(s string) string {
	return labelValueEscaper.Replace(strings.ToValidUTF8(s, "\ufffd"))
}

func escapeHelp(s string) string {
	return helpEscaper.Replace(strings.ToValidUTF8(s, "\ufffd"))
}

func formatValue(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsNaN(v):
		return "NaN"
	default:
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
}

func writeHeader(b *strings.Builder, name, help, kind string) {
	b.WriteString("# HELP " + name + " " + escapeHelp(help) + "\n")
	b.WriteString("# TYPE " + name + " " + kind + "\n")
}

// writeSample writes one sample line. extraName/extraValue add a trailing
// label (the histogram "le") without copying the label slices.
func writeSample(b *strings.Builder, name string, labels, values []string, extraName, extraValue string, v float64) {
	b.WriteString(name)
	if len(labels) > 0 || extraName != "" {
		b.WriteByte('{')
		for i, l := range labels {
			writeLabel(b, i > 0, l, values[i])
		}
		if extraName != "" {
			writeLabel(b, len(labels) > 0, extraName, extraValue)
		}
		b.WriteByte('}')
	}
	b.WriteString(" " + formatValue(v) + "\n")
}

func writeLabel(b *strings.Builder, comma bool, name, value string) {
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(name + `="` + escapeLabelValue(value) + `"`)
}
