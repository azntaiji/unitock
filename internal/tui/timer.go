package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/task"
)

// theme is what the view needs to know about the terminal to pick colours (D-09).
type theme struct {
	dark    bool // background is dark; picks the dark half of each adaptive pair
	noColor bool // NO_COLOR or a terminal without colour: no colours, text badge instead
}

// band is the colour band for the remaining fraction of the estimate (R-18, D-09).
type band int

const (
	bandNeutral  band = iota // more than 50% left
	bandInfo                 // 50% or less
	bandWarning              // 25% or less
	bandError                // 10% or less
	bandOvertime             // estimate used up; error colour
)

// label names the band in the NO_COLOR text badge (D-09).
func (b band) label() string {
	return [...]string{"on track", "half left", "quarter left", "almost out", "overtime"}[b]
}

// bandAt is t's band at now. A task with no time left but not yet past its estimate is
// in the error band, not overtime.
func bandAt(t task.Task, now time.Time) band {
	if t.Overtime(now) > 0 {
		return bandOvertime
	}
	est := t.Estimate()
	if est <= 0 {
		return bandError
	}
	// Compare remaining*100 against est*pct in integer nanoseconds to avoid float edges.
	rem := t.Remaining(now)
	switch {
	case rem*10 <= est:
		return bandError
	case rem*4 <= est:
		return bandWarning
	case rem*2 <= est:
		return bandInfo
	}
	return bandNeutral
}

// color is the band's foreground, or no colour for neutral and under NO_COLOR. Each
// colour is a light/dark pair chosen by the terminal background (D-09).
func (b band) color(th theme) color.Color {
	if th.noColor {
		return lipgloss.NoColor{}
	}
	ld := lipgloss.LightDark(th.dark)
	switch b {
	case bandInfo:
		return ld(lipgloss.Color("#0969DA"), lipgloss.Color("#58A6FF"))
	case bandWarning:
		return ld(lipgloss.Color("#9A6700"), lipgloss.Color("#D29922"))
	case bandError, bandOvertime:
		return ld(lipgloss.Color("#CF222E"), lipgloss.Color("#F85149"))
	}
	return lipgloss.NoColor{}
}

// blockMinWidth is the terminal width below which the timer falls back to plain bold
// text (R-17, D-09). Zero width means not yet known, and uses block digits.
const blockMinWidth = 60

// glyphs is the 5-row block font for the timer: digits, ':' and '+'.
var glyphs = map[rune][5]string{
	'0': {"█████", "█   █", "█   █", "█   █", "█████"},
	'1': {"   █ ", "  ██ ", "   █ ", "   █ ", "  ███"},
	'2': {"█████", "    █", "█████", "█    ", "█████"},
	'3': {"█████", "    █", " ████", "    █", "█████"},
	'4': {"█   █", "█   █", "█████", "    █", "    █"},
	'5': {"█████", "█    ", "█████", "    █", "█████"},
	'6': {"█████", "█    ", "█████", "█   █", "█████"},
	'7': {"█████", "    █", "   █ ", "  █  ", "  █  "},
	'8': {"█████", "█   █", "█████", "█   █", "█████"},
	'9': {"█████", "█   █", "█████", "    █", "█████"},
	':': {" ", "█", " ", "█", " "},
	'+': {"     ", "  █  ", "█████", "  █  ", "     "},
}

// blockText renders s in the block font, one space between glyphs. Characters without
// a glyph are skipped.
func blockText(s string) string {
	var rows [5][]string
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for i := range rows {
			rows[i] = append(rows[i], g[i])
		}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.Join(r, " ")
	}
	return strings.Join(out, "\n")
}

// barWidth is the progress bar's width in cells, excluding the percentage.
const barWidth = 40

// progressBar draws elapsed / estimate as a bar plus the elapsed percentage. Overtime
// fills the bar; the percentage keeps counting past 100 (R-17, R-18).
func progressBar(t task.Task, now time.Time, fg color.Color) string {
	est, el := t.Estimate(), t.Elapsed(now)
	pct := 100
	if est > 0 {
		pct = int(el * 100 / est)
	}
	filled := min(pct*barWidth/100, barWidth)
	bar := lipgloss.NewStyle().Foreground(fg).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Faint(true).Render(strings.Repeat("░", barWidth-filled))
	return fmt.Sprintf("%s %3d%%", bar, pct)
}

// badge is the status badge: RUNNING or PAUSED, plus the band's name under NO_COLOR
// (R-17, D-09).
func badge(t task.Task, b band, th theme) string {
	s := "● RUNNING"
	if t.Paused() {
		s = "|| PAUSED"
	}
	if th.noColor {
		return "[" + s + " · " + strings.ToUpper(b.label()) + "]"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(b.color(th)).Render(s)
}
