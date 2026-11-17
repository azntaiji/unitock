package tui

import "charm.land/lipgloss/v2"

var hintStyle = lipgloss.NewStyle().Faint(true)

// renderHints draws the key-hint bar shown on every screen (R-07).
func renderHints(s string) string { return hintStyle.Render(s) }
