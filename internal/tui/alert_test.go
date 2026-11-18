package tui

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/unitock/internal/task"
)

// runOut starts a 1-minute task and ticks at zero, so the alert fires.
func (h *harness) runOut() {
	h.t.Helper()
	h.startTask("write", "1m")
	h.clock.Advance(time.Minute)
	h.tick()
	if !h.m.active.alert.open {
		h.t.Fatalf("alert did not open at zero:\n%s", h.view())
	}
}

// P2-05: at zero the modal opens, one notification is sent, and the sound starts and
// keeps going across ticks (R-30).
func TestAlertFiresOnceAtZero(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "1m")
	h.clock.Advance(59 * time.Second)
	h.tick()
	if h.m.active.alert.open || h.alarm.playing {
		t.Fatal("alert fired before zero")
	}
	h.clock.Advance(time.Second)
	h.tick()
	v := h.view()
	if !h.m.active.alert.open || !strings.Contains(v, "Time's up") || !strings.Contains(v, "Mark as completed") ||
		!strings.Contains(v, "d extend") {
		t.Fatalf("no alert at zero:\n%s", v)
	}
	for range 5 {
		h.clock.Advance(time.Second)
		h.tick()
	}
	if !slices.Equal(h.alarm.notified, []string{"write"}) || h.alarm.starts != 1 || !h.alarm.playing {
		t.Errorf("notified %v, starts %d, playing %v; want one notification, sound on", h.alarm.notified, h.alarm.starts, h.alarm.playing)
	}
}

// Mark as completed (e, or enter after choosing it) stops the sound and completes.
func TestAlertComplete(t *testing.T) {
	for _, keys := range [][]string{{"e"}, {"right", "enter"}, {"tab", "enter"}} {
		t.Run(strings.Join(keys, "+"), func(t *testing.T) {
			h := newHarness(t)
			h.runOut()
			h.press(keys...)
			if h.alarm.playing || h.m.screen != screenDone {
				t.Fatalf("playing %v screen %d", h.alarm.playing, h.m.screen)
			}
			got, err := h.store.Get(context.Background(), h.m.done.task.ID)
			if err != nil || got.Status != task.Completed {
				t.Errorf("stored %+v, %v", got, err)
			}
		})
	}
}

// Extend (d, or enter on the default choice) stops the sound and opens the input;
// esc returns to the alert with the sound on again but no second notification (R-32).
func TestAlertExtendAndEscBack(t *testing.T) {
	for _, k := range []string{"d", "enter"} {
		t.Run(k, func(t *testing.T) {
			h := newHarness(t)
			h.runOut()
			h.press(k)
			if h.alarm.playing || h.m.active.alert.open || !h.m.active.ext.open || !strings.Contains(h.view(), "esc back") {
				t.Fatalf("after %s: playing %v alert %v ext %v", k, h.alarm.playing, h.m.active.alert.open, h.m.active.ext.open)
			}
			h.press("esc")
			if !h.m.active.alert.open || !h.alarm.playing || len(h.alarm.notified) != 1 {
				t.Errorf("esc: alert %v playing %v notified %v", h.m.active.alert.open, h.alarm.playing, h.alarm.notified)
			}
		})
	}
}

// After extending from the alert, the alert fires again when the new time runs out.
func TestAlertRefiresAfterExtend(t *testing.T) {
	h := newHarness(t)
	h.runOut()
	h.press("d")
	h.typeText("1m")
	h.press("enter")
	if h.m.active.alert.open || h.m.active.ext.open {
		t.Fatal("alert or input still open after extending")
	}
	h.clock.Advance(time.Minute)
	h.tick()
	if !h.m.active.alert.open || len(h.alarm.notified) != 2 {
		t.Errorf("second time-up: alert %v notified %v", h.m.active.alert.open, h.alarm.notified)
	}
}

// Other screen keys are inactive while the alert is open; q quits and stops the sound
// (R-31).
func TestAlertOtherKeysInactiveAndQuit(t *testing.T) {
	h := newHarness(t)
	h.runOut()
	h.press("p", "n")
	if h.m.active.task.Paused() || h.m.active.conf.open || !h.m.active.alert.open {
		t.Fatalf("keys acted under the alert: paused %v confirm %v", h.m.active.task.Paused(), h.m.active.conf.open)
	}
	h.press("q")
	if !h.quit || h.alarm.playing {
		t.Errorf("q: quit %v playing %v", h.quit, h.alarm.playing)
	}
}

func TestAlertCtrlCStopsSound(t *testing.T) {
	h := newHarness(t)
	h.runOut()
	h.press("ctrl+c")
	if !h.quit || h.alarm.playing {
		t.Errorf("ctrl+c: quit %v playing %v", h.quit, h.alarm.playing)
	}
}

// A task that ran out while the app was closed alerts at launch (R-05); a paused one
// does not until it is resumed.
func TestAlertOnRelaunchInOvertime(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "1m")
	h.press("q")
	h.clock.Advance(10 * time.Minute)
	alarm := &fakeAlarm{}
	d := h.m.deps
	d.Alarm = alarm
	m, err := New(d)
	if err != nil {
		t.Fatal(err)
	}
	if !m.active.alert.open || !alarm.playing || len(alarm.notified) != 1 {
		t.Errorf("relaunch in overtime: alert %v playing %v notified %v", m.active.alert.open, alarm.playing, alarm.notified)
	}
}

func TestAlertWaitsWhilePaused(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "1m")
	h.clock.Advance(30 * time.Second)
	h.press("p")
	h.clock.Advance(time.Hour)
	h.tick()
	if h.m.active.alert.open {
		t.Fatal("alert fired while paused")
	}
	h.press("p")
	h.clock.Advance(30 * time.Second)
	h.tick()
	if !h.m.active.alert.open {
		t.Error("alert did not fire after resuming to zero")
	}
}

// The alert floats over the Active screen (R-30): the screen keeps its height, and the
// box covers the middle of it instead of being added below.
func TestAlertFloatsOverScreen(t *testing.T) {
	tk := task.Task{ID: 1, Name: "Write report", Project: "unitock", Status: task.Active, StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	m := newActiveModel(tk, 1)
	now := t0.Add(25 * time.Minute)
	under := m.view(now, 80, theme{})
	m.alert.open = true
	over := m.view(now, 80, theme{})
	if lipgloss.Height(over) != lipgloss.Height(under) {
		t.Errorf("height %d with alert, %d without; the box must not add lines", lipgloss.Height(over), lipgloss.Height(under))
	}
	lines := strings.Split(ansi.Strip(over), "\n")
	if !strings.Contains(lines[0], "OVERTIME") && !strings.Contains(lines[0], "ENDING") {
		t.Errorf("badge line covered: %q", lines[0])
	}
	if last := lines[len(lines)-1]; !strings.Contains(last, "Estimate") {
		t.Errorf("last line covered: %q", last)
	}
	mid := strings.Join(lines[1:len(lines)-1], "\n")
	if !strings.Contains(mid, "Time's up") || !strings.Contains(mid, "Mark as completed") {
		t.Errorf("box not drawn over the middle:\n%s", ansi.Strip(over))
	}
	t.Logf("\n%s", ansi.Strip(over))
}
