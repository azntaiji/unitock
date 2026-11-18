package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/stats"
)

// Chart glyphs. Estimated and actual differ in shape as well as colour, so the chart
// reads under NO_COLOR (D-09).
const (
	estGlyph = "▒"
	actGlyph = "█"
)

// Chart size limits. The plot is as tall as the terminal allows between these; at the
// 60×20 minimum (R-08) it gets chartMinRows.
const (
	chartMinRows   = 4
	chartMaxRows   = 16
	chartDefRows   = 8  // terminal size not yet known
	chartDefWidth  = 60 // terminal size not yet known
	chartMaxSlot   = 16 // widest column per bucket
	statsFixedRows = 14 // Stats lines outside the plot, plus the hint bar
)

var estStyle = lipgloss.NewStyle().Faint(true)

func actStyle(th theme) lipgloss.Style {
	if th.noColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Blue)
}

// chartRows is the plot height for a terminal height (0 when unknown).
func chartRows(height int) int {
	if height <= 0 {
		return chartDefRows
	}
	return min(max(height-statsFixedRows, chartMinRows), chartMaxRows)
}

// renderChart draws the R-43 chart: a legend, a y-axis in hours and minutes, an
// estimated and an actual bar per bucket, and each bucket's label beneath; or the R-45
// empty state when nothing was completed in the period.
func renderChart(s stats.Summary, width, height int, th theme) string {
	if width <= 0 {
		width = chartDefWidth
	}
	rows := chartRows(height)
	legend := estStyle.Render(estGlyph+estGlyph) + " Estimated  " + actStyle(th).Render(actGlyph+actGlyph) + " Actual"
	if s.Empty() {
		lines := []string{legend}
		msg := dimStyle.Render("No completed tasks in this period.")
		for i := range rows + 2 {
			if i == rows/2 {
				lines = append(lines, "  "+msg)
			} else {
				lines = append(lines, "")
			}
		}
		return strings.Join(lines, "\n")
	}

	var peak time.Duration
	for _, b := range s.Buckets {
		peak = max(peak, b.Estimated, b.Actual)
	}
	top := niceTop(peak)
	labels := map[int]string{0: formatHM(top), rows / 2: formatHM(top / 2), rows: "0"}
	if rows%2 == 1 { // keep the middle label on the row that is exactly half
		labels[rows/2] = ""
	}
	axisW := 0
	for _, l := range labels {
		axisW = max(axisW, len(l))
	}

	n := len(s.Buckets)
	slot := min(max((width-axisW-1)/n, 4), chartMaxSlot)
	bar := 1
	if slot >= 7 {
		bar = 2
	}
	est := make([]int, n)
	act := make([]int, n)
	for i, b := range s.Buckets {
		est[i], act[i] = barCells(b.Estimated, top, rows), barCells(b.Actual, top, rows)
	}

	var out []string
	out = append(out, legend)
	pad := (slot - 2*bar) / 2
	for r := range rows {
		level := rows - r // cells from the bottom this row needs to be filled
		var line strings.Builder
		fmt.Fprintf(&line, "%*s", axisW, labels[r])
		if labels[r] != "" {
			line.WriteString("┤")
		} else {
			line.WriteString("│")
		}
		for i := range n {
			line.WriteString(strings.Repeat(" ", pad))
			line.WriteString(cell(est[i] >= level, bar, estGlyph, estStyle))
			line.WriteString(cell(act[i] >= level, bar, actGlyph, actStyle(th)))
			line.WriteString(strings.Repeat(" ", slot-pad-2*bar))
		}
		out = append(out, strings.TrimRight(line.String(), " "))
	}
	out = append(out, fmt.Sprintf("%*s", axisW, "0")+"└"+strings.Repeat("─", n*slot))

	// x-axis labels, centred under each bucket; split at the first space onto two lines
	// when a label does not fit its column (7 days at narrow widths).
	first, second := make([]string, n), make([]string, n)
	split := false
	for _, b := range s.Buckets {
		if len(b.Label) > slot-1 {
			split = true
		}
	}
	for i, b := range s.Buckets {
		first[i] = b.Label
		if split {
			first[i], second[i], _ = strings.Cut(b.Label, " ")
		}
	}
	out = append(out, labelRow(first, axisW+1, slot))
	if split {
		out = append(out, labelRow(second, axisW+1, slot))
	}
	return strings.Join(out, "\n")
}

func cell(filled bool, w int, glyph string, st lipgloss.Style) string {
	if !filled {
		return strings.Repeat(" ", w)
	}
	return st.Render(strings.Repeat(glyph, w))
}

// barCells is how many rows a bar of d fills on a plot of rows rows topped at top:
// rounded to the nearest row, and at least one for any time at all.
func barCells(d, top time.Duration, rows int) int {
	if d <= 0 {
		return 0
	}
	c := int((int64(d)*int64(rows) + int64(top)/2) / int64(top))
	return min(max(c, 1), rows)
}

func labelRow(labels []string, indent, slot int) string {
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", indent))
	for _, l := range labels {
		w := len([]rune(l))
		left := max((slot-w)/2, 0)
		b.WriteString(strings.Repeat(" ", left) + l + strings.Repeat(" ", max(slot-left-w, 0)))
	}
	return strings.TrimRight(b.String(), " ")
}

// niceTop is the y-axis top: the smallest of 10m, 20m, 30m, 1h, 2h, 3h, 4h, 6h, 8h, 12h,
// 16h, 24h, then doubling 16h and 24h in turn, that is at least d. Each halves to whole
// minutes, so the middle label is exact.
func niceTop(d time.Duration) time.Duration {
	steps := []time.Duration{10 * time.Minute, 20 * time.Minute, 30 * time.Minute,
		time.Hour, 2 * time.Hour, 3 * time.Hour, 4 * time.Hour, 6 * time.Hour,
		8 * time.Hour, 12 * time.Hour, 16 * time.Hour, 24 * time.Hour}
	for _, s := range steps {
		if d <= s {
			return s
		}
	}
	a, b := 16*time.Hour, 24*time.Hour
	for {
		a, b = 2*a, 2*b
		if d <= a {
			return a
		}
		if d <= b {
			return b
		}
	}
}

// formatHM renders an axis value in hours and minutes: 45m, 2h, 1h30m (R-43).
func formatHM(d time.Duration) string {
	h, m := int(d/time.Hour), int(d%time.Hour/time.Minute)
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
