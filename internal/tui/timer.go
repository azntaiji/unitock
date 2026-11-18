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
// Colours are the terminal's own ANSI palette, so they follow its light or dark theme.
type theme struct {
	noColor bool // NO_COLOR or a terminal without colour: no colours, reverse-video badge
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
	// Integer comparisons avoid float edges at exactly 10%, 25%, and 50%.
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

// color is the band's ANSI foreground for the digits and bar, or none for neutral and
// under NO_COLOR (D-09).
func (b band) color(th theme) color.Color {
	if th.noColor {
		return lipgloss.NoColor{}
	}
	switch b {
	case bandInfo:
		return lipgloss.Blue
	case bandWarning:
		return lipgloss.Yellow
	case bandError, bandOvertime:
		return lipgloss.Red
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

// badge is the status badge (R-17, D-09): bold black text on an ANSI background, or
// reverse video under NO_COLOR. PAUSED overrides the others; ENDING covers 25% or less
// left (the warning and error bands).
func badge(t task.Task, b band, th theme) string {
	text, bg := "● RUNNING", color.Color(lipgloss.Cyan)
	switch {
	case t.Paused():
		text, bg = "❚❚ PAUSED", lipgloss.White
	case b == bandOvertime:
		text, bg = "▲ OVERTIME", lipgloss.Red
	case b >= bandWarning:
		text, bg = "◐ ENDING", lipgloss.Yellow
	}
	st := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	if th.noColor {
		return st.Reverse(true).Render(text)
	}
	return st.Foreground(lipgloss.Black).Background(bg).Render(text)
}

// heading is the lines beneath the badge: the project in bold, then the task name
// dimmed; or the task name alone in bold when there is no project (R-17).
func heading(t task.Task) []string {
	if t.Project == "" {
		return []string{lipgloss.NewStyle().Bold(true).Render(t.Name)}
	}
	return []string{
		lipgloss.NewStyle().Bold(true).Render(t.Project),
		lipgloss.NewStyle().Faint(true).Render(t.Name),
	}
}
