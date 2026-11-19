package tui

import (
	"strings"
	"testing"
	"time"
)

// P2-02: pausing freezes the timer, shows PAUSED, and p resumes; paused time is not
// counted as elapsed (R-20).
func TestPauseFreezesTimerAndResumes(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.clock.Advance(5 * time.Minute)
	h.press("p")
	if !h.m.active.task.Paused() || !strings.Contains(h.view(), "PAUSED") || !strings.Contains(h.view(), "p resume") {
		t.Fatalf("not paused:\n%s", h.view())
	}
	h.clock.Advance(30 * time.Minute)
	h.tick()
	if !strings.Contains(h.view(), "Elapsed 05:00") { // 20:00 left
		t.Errorf("timer not frozen at 20:00 while paused:\n%s", h.view())
	}
	h.press("p")
	if h.m.active.task.Paused() || !strings.Contains(h.view(), "RUNNING") || !strings.Contains(h.view(), "p pause") {
		t.Fatalf("not resumed:\n%s", h.view())
	}
	h.clock.Advance(time.Minute)
	h.tick()
	if !strings.Contains(h.view(), "Elapsed 06:00") { // 19:00 left
		t.Errorf("after resume + 1m:\n%s", h.view())
	}
	st, err := h.store.Get(h.m.deps.Ctx, h.m.active.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Pauses) != 1 || st.Pauses[0].EndedAt.Sub(st.Pauses[0].StartedAt) != 30*time.Minute {
		t.Errorf("stored pauses = %+v, want one 30m pause", st.Pauses)
	}
}

// A task paused at quit is still paused at launch, with the time while closed not
// counted (R-05, P2-02).
func TestPauseSurvivesRelaunch(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.clock.Advance(5 * time.Minute)
	h.press("p", "ctrl+c")
	if !h.quit {
		t.Fatal("ctrl+c did not quit")
	}
	h.clock.Advance(2 * time.Hour)
	m, err := New(h.m.deps)
	if err != nil {
		t.Fatal(err)
	}
	if v := m.View().Content; m.screen != screenActive || !strings.Contains(v, "PAUSED") || !strings.Contains(v, "Elapsed 05:00") {
		t.Errorf("relaunch view:\n%s", v)
	}
}

func TestPauseErrorStays(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	_ = h.store.Close()
	h.press("p")
	if h.m.active.task.Paused() || !strings.Contains(h.m.active.err, "Could not save") || h.m.active.busy {
		t.Errorf("paused=%v err=%q busy=%v", h.m.active.task.Paused(), h.m.active.err, h.m.active.busy)
	}
}
