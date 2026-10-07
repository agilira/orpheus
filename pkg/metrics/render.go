// render.go: the exposition of a registry.
//
// WHY: output is sorted by metric name and label values so that two scrapes
// of the same state are byte-identical, which keeps diffs and tests honest.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package metrics

import (
	"maps"
	"slices"
	"strings"
)

func (r *registry) render() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	b.Grow(r.nbytes + 1024)
	for _, name := range slices.Sorted(maps.Keys(r.families)) {
		writeFamily(&b, r.families[name])
	}
	writeDropped(&b, r.dropped)
	return b.String()
}

func writeFamily(b *strings.Builder, f *family) {
	writeHeader(b, f.name, f.help, kindNames[f.kind])
	all := slices.Collect(maps.Values(f.series))
	slices.SortFunc(all, func(x, y *series) int { return slices.Compare(x.values, y.values) })
	for _, s := range all {
		if f.kind == histogramKind {
			writeHistogram(b, f, s)
		} else {
			writeSample(b, f.name, f.labels, s.values, "", "", s.value)
		}
	}
}

func writeHistogram(b *strings.Builder, f *family, s *series) {
	var cumulative uint64
	for i, bound := range f.buckets {
		cumulative += s.counts[i]
		writeSample(b, f.name+"_bucket", f.labels, s.values, "le", formatValue(bound), float64(cumulative))
	}
	writeSample(b, f.name+"_bucket", f.labels, s.values, "le", "+Inf", float64(s.count))
	writeSample(b, f.name+"_sum", f.labels, s.values, "", "", s.value)
	writeSample(b, f.name+"_count", f.labels, s.values, "", "", float64(s.count))
}

// writeDropped always writes the header, so that an alert on the metric
// can tell "nothing dropped" from "collector missing".
func writeDropped(b *strings.Builder, dropped map[DropReason]uint64) {
	writeHeader(b, droppedName, droppedHelp, "counter")
	label := []string{"reason"}
	for _, reason := range slices.Sorted(maps.Keys(dropped)) {
		writeSample(b, droppedName, label, []string{string(reason)}, "", "", float64(dropped[reason]))
	}
}
