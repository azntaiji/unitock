package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var hintStyle = lipgloss.NewStyle().Faint(true)

// renderHints draws the key-hint bar shown on every screen (R-07).
func renderHints(s string) string { return hintStyle.Render(s) }

// chordHints is the hint line for the app-wide ctrl chords (R-06, R-07): the given ones,
// then quit.
func chordHints(chords ...string) string {
	return strings.Join(append(chords, "ctrl+c quit"), " · ")
}
