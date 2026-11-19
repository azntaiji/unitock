package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/stats"
)

// pressS sends `s` and, if it opened Stats, runs the data load as the runtime would.
func (h *harness) pressS() {
	h.t.Helper()
	next, cmd := h.m.Update(key("s"))
	h.m = next.(Model)
	if h.m.screen == screenStats && cmd != nil {
		h.send(cmd())
	}
}

// P3-04: `s` on Start works only with the button focused (R-14); in an input it types.
func TestStatsFromStartButtonOnly(t *testing.T) {
	h := newHarness(t)
	h.pressS()
	if h.m.screen != screenStart || h.m.start.project.Value() != "s" {
		t.Fatalf("s in the Project input: screen %d, value %q; want typed text", h.m.screen, h.m.start.project.Value())
	}
	h.press("esc") // focus the button
	before := h.m.start
	h.pressS()
	if h.m.screen != screenStats || !h.m.stats.loaded {
		t.Fatalf("s on the button: screen %d loaded %v; want Stats", h.m.screen, h.m.stats.loaded)
	}
	if !strings.Contains(h.view(), "tab period") {
		t.Errorf("Stats hints missing:\n%s", h.view())
	}
	h.press("esc")
	if h.m.screen != screenStart || h.m.start.focus != before.focus || h.m.start.project.Value() != "s" {
		t.Errorf("esc did not return to Start as left: screen %d focus %d value %q",
			h.m.screen, h.m.start.focus, h.m.start.project.Value())
	}
}

// P3-04: tab cycles 7 days → 30 days → 3 months → 7 days, and each opening starts on
// 7 days (R-40).
func TestStatsTabCyclesAndReopensOnSevenDays(t *testing.T) {
	h := newHarness(t)
	h.press("esc")
	h.pressS()
	var got []stats.Period
	for range 4 {
		got = append(got, h.m.stats.period)
		h.press("tab")
	}
	want := []stats.Period{stats.Days7, stats.Days30, stats.Months3, stats.Days7}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("periods = %v, want %v", got, want)
		}
	}
	h.press("tab", "esc")
	h.pressS()
	if h.m.stats.period != stats.Days7 {
		t.Errorf("reopened on %v, want 7 days", h.m.stats.period)
	}
}

// P3-04: Stats opens from Active, the timer keeps running underneath, and esc returns
// to Active with the time moved on (R-24, R-46).
func TestStatsFromActiveTimerKeepsRunning(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.pressS()
	if h.m.screen != screenStats {
		t.Fatalf("s on Active did not open Stats")
	}
	for range 90 {
		h.clock.Advance(time.Second)
		h.tick()
	}
	h.press("esc")
	if h.m.screen != screenActive {
		t.Fatalf("esc returned to screen %d, want Active", h.m.screen)
	}
	if v := h.view(); !strings.Contains(v, "Elapsed 01:30") { // 23:30 left
		t.Errorf("timer did not keep running under Stats:\n%s", v)
	}
	// The tick loop is the same one, not a second loop.
	if h.m.active.gen != h.m.gen {
		t.Errorf("active gen %d, root gen %d", h.m.active.gen, h.m.gen)
	}
}

// `s` does nothing while Active has an input or confirmation open.
func TestStatsNotFromActiveDialogs(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.press("n") // cancel confirmation
	h.pressS()
	if h.m.screen != screenActive {
		t.Errorf("s with the cancel confirmation open opened screen %d", h.m.screen)
	}
	h.press("esc", "d")
	h.pressS()
	if h.m.screen != screenActive || h.m.active.ext.input.Value() != "s" {
		t.Errorf("s in the duration input: screen %d value %q; want typed", h.m.screen, h.m.active.ext.input.Value())
	}
}

// P3-04: the alert fires over Stats, takes the keys, and Mark as completed works from
// there (R-30, R-31).
func TestStatsAlertOverlayComplete(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "1m")
	h.pressS()
	h.clock.Advance(time.Minute)
	h.tick()
	v := h.view()
	if h.m.screen != screenStats || !h.m.active.alert.open || !h.alarm.playing {
		t.Fatalf("alert over Stats: screen %d open %v playing %v", h.m.screen, h.m.active.alert.open, h.alarm.playing)
	}
	if !strings.Contains(v, "Time's up") || !strings.Contains(v, "d extend") {
		t.Errorf("alert not drawn over Stats:\n%s", v)
	}
	h.press("tab", "esc") // tab chooses within the alert; esc is inactive
	if h.m.stats.period != stats.Days7 || h.m.screen != screenStats || !h.m.active.alert.open {
		t.Errorf("keys leaked past the alert: period %v screen %d", h.m.stats.period, h.m.screen)
	}
	h.press("e")
	if h.m.screen != screenDone || h.alarm.playing {
		t.Errorf("e on the alert over Stats: screen %d playing %v; want Done, silent", h.m.screen, h.alarm.playing)
	}
}

// Extend from the alert over Stats switches to Active, where the duration input is.
func TestStatsAlertOverlayExtend(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "1m")
	h.pressS()
	h.clock.Advance(time.Minute)
	h.tick()
	h.press("d")
	if h.m.screen != screenActive || !h.m.active.ext.open || h.alarm.playing {
		t.Errorf("Extend over Stats: screen %d input %v playing %v; want Active input, silent",
			h.m.screen, h.m.active.ext.open, h.alarm.playing)
	}
}

func TestStatsQuit(t *testing.T) {
	h := newHarness(t)
	h.press("esc")
	h.pressS()
	h.press("q")
	if !h.quit {
		t.Error("q on Stats did not quit")
	}
}
