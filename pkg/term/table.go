// table.go: aligned columns for terminal output.
//
// WHY: the cells of a status table are names chosen by someone else. Every
// cell is sanitized and measured after sanitizing, so a cell can neither add
// a line nor shift the columns that follow it. Color is applied after the
// width is computed, so styled and plain tables have the same layout.
//
// Copyright (c) 2025 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package term

import (
	"io"
	"strings"
)

// columnGap separates columns. Two spaces keep columns readable without
// box-drawing characters, which not every font renders.
const columnGap = "  "

// Cell is one table cell. Text may contain anything; it is sanitized.
type Cell struct {
	Text  string
	Style Style
}

// Text returns an unstyled cell.
func Text(s string) Cell { return Cell{Text: s} }

// Styled returns a cell rendered with style when color is available.
func Styled(style Style, s string) Cell { return Cell{Text: s, Style: style} }

// Table collects rows and renders them with aligned columns.
// Rows may have any number of cells; no cell is ever dropped.
type Table interface {
	Row(cells ...Cell)
	Render(w io.Writer) error
}

type table struct {
	caps Caps
	rows [][]Cell
}

// NewTable returns a table for a writer with the given capabilities.
// Headers, when present, form the first row and are rendered bold.
func NewTable(caps Caps, headers ...string) Table {
	t := &table{caps: caps}
	if len(headers) > 0 {
		row := make([]Cell, len(headers))
		for i, h := range headers {
			row[i] = Styled(Bold, h)
		}
		t.rows = append(t.rows, row)
	}
	return t
}

func (t *table) Row(cells ...Cell) {
	t.rows = append(t.rows, append([]Cell(nil), cells...))
}

// Render writes the whole table with a single Write, so a failure cannot
// leave half a table on screen without being reported.
func (t *table) Render(w io.Writer) error {
	if len(t.rows) == 0 {
		return nil
	}
	safe := t.sanitized()
	widths := columnWidths(safe)
	var b strings.Builder
	for i, row := range safe {
		t.writeRow(&b, t.rows[i], row, widths)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func (t *table) sanitized() [][]string {
	out := make([][]string, len(t.rows))
	for i, row := range t.rows {
		out[i] = make([]string, len(row))
		for j, c := range row {
			out[i][j] = Sanitize(c.Text)
		}
	}
	return out
}

func columnWidths(rows [][]string) []int {
	var widths []int
	for _, row := range rows {
		for j, s := range row {
			if j == len(widths) {
				widths = append(widths, 0)
			}
			widths[j] = max(widths[j], Width(s))
		}
	}
	return widths
}

// writeRow pads every cell but the last of its row, so lines never end in
// spaces. safe holds the already sanitized text of cells.
func (t *table) writeRow(b *strings.Builder, cells []Cell, safe []string, widths []int) {
	for j, s := range safe {
		if j > 0 {
			b.WriteString(columnGap)
		}
		b.WriteString(Paint(t.caps, cells[j].Style, s))
		if j < len(safe)-1 {
			b.WriteString(strings.Repeat(" ", widths[j]-Width(s)))
		}
	}
	b.WriteByte('\n')
}
