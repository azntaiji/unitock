package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Below 60×20 the app shows the too-small message instead of the screen, ignores the
// screen's keys, and redraws normally once resized (R-08).
func TestTooSmallShowsMessageAndRecovers(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 59, Height: 30}, {Width: 100, Height: 19}} {
		h := newHarness(t)
		h.send(size)
		v := h.view()
		if !strings.Contains(v, "Terminal too small") || strings.Contains(v, "Task name") {
			t.Fatalf("%dx%d: view\n%s", size.Width, size.Height, v)
		}
		if !strings.Contains(v, "60×20") || !strings.Contains(v, "ctrl+c quit") {
			t.Errorf("%dx%d: message lacks the minimum or the quit hint:\n%s", size.Width, size.Height, v)
		}
		h.typeText("q")
		h.press("tab")
		if h.quit || h.m.start.project.Value() != "" || h.m.start.focus != focusProject {
			t.Errorf("%dx%d: keys reached the hidden screen (quit %v, value %q, focus %d)",
				size.Width, size.Height, h.quit, h.m.start.project.Value(), h.m.start.focus)
		}
		h.send(tea.WindowSizeMsg{Width: MinWidth, Height: MinHeight})
		if v := h.view(); strings.Contains(v, "too small") || !strings.Contains(v, "Task name") {
			t.Errorf("at the minimum size the screen is not shown:\n%s", v)
		}
	}
}

func TestTooSmallCtrlCQuits(t *testing.T) {
	h := newHarness(t)
	h.send(tea.WindowSizeMsg{Width: 40, Height: 10})
	h.press("ctrl+c")
	if !h.quit {
		t.Error("ctrl+c did not quit while too small")
	}
}

// The timer and alert keep working behind the message: time up still fires the alarm.
func TestTooSmallAlertStillFires(t *testing.T) {
	h := newHarness(t)
	h.runOut() // 1m task, ticked at zero
	h.send(tea.WindowSizeMsg{Width: 40, Height: 10})
	if !h.m.active.alert.open || !h.alarm.playing {
		t.Fatal("alert not open with the alarm playing")
	}
	if v := h.view(); !strings.Contains(v, "too small") {
		t.Errorf("view\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	if v := h.view(); !strings.Contains(v, "Time's up") {
		t.Errorf("alert not shown after resize:\n%s", v)
	}
}
