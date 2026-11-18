package tui

import (
	"strings"
	"testing"
	"time"
)

// P2-03: d opens a duration input; enter adds exactly the entered amount (R-21).
func TestExtendAddsEnteredAmount(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.clock.Advance(5 * time.Minute)
	h.press("d")
	if !h.m.active.ext.open || !strings.Contains(h.view(), "Add time") || !strings.Contains(h.view(), "enter add") {
		t.Fatalf("d did not open the input:\n%s", h.view())
	}
	h.typeText("10 min")
	h.press("enter")
	if h.m.active.ext.open {
		t.Fatalf("input still open: %q %q", h.m.active.ext.err, h.m.active.err)
	}
	if v := h.view(); !strings.Contains(v, "Remaining 30:00 · Estimate 35:00") {
		t.Errorf("after extend:\n%s", v)
	}
	st, err := h.store.Get(h.m.deps.Ctx, h.m.active.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Extensions) != 1 || st.Extensions[0].Added != 10*time.Minute || !st.Extensions[0].At.Equal(t0.Add(5*time.Minute)) {
		t.Errorf("stored extensions = %+v", st.Extensions)
	}
}

func TestExtendWhilePaused(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.press("p")
	h.press("d")
	h.typeText("1:00")
	h.press("enter")
	if !h.m.active.task.Paused() || h.m.active.task.Estimate() != 85*time.Minute {
		t.Errorf("paused=%v estimate=%v, want still paused with 1h25m", h.m.active.task.Paused(), h.m.active.task.Estimate())
	}
}

// esc closes unchanged; while open, q p e are text, not commands (R-06, R-21).
func TestExtendEscAndTypingKeys(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.press("d")
	h.typeText("qpe")
	if h.quit || h.m.active.task.Paused() || h.m.screen != screenActive || h.m.active.ext.input.Value() != "qpe" {
		t.Fatalf("keys acted as commands: quit=%v paused=%v screen=%d value=%q",
			h.quit, h.m.active.task.Paused(), h.m.screen, h.m.active.ext.input.Value())
	}
	h.press("esc")
	if h.m.active.ext.open || h.m.active.task.Estimate() != 25*time.Minute || len(h.m.active.task.Extensions) != 0 {
		t.Errorf("esc: open=%v estimate=%v", h.m.active.ext.open, h.m.active.task.Estimate())
	}
	if !strings.Contains(h.view(), "d add time") {
		t.Errorf("hints after esc:\n%s", h.view())
	}
}

func TestExtendInvalidShowsError(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.press("d")
	h.typeText("soon")
	h.press("enter")
	if !h.m.active.ext.open || !strings.Contains(h.view(), "1h30m") || len(h.m.active.task.Extensions) != 0 {
		t.Errorf("invalid input: open=%v view:\n%s", h.m.active.ext.open, h.view())
	}
}

// From the alert (D-10, R-32), the overtime is added too so remaining equals the entry.
func TestExtendFromAlertSetsRemaining(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.clock.Advance(28 * time.Minute) // 3m overtime
	h.tick()
	if !h.m.active.alert.open {
		t.Fatal("alert did not open")
	}
	h.press("enter") // Extend is selected first
	if !h.m.active.ext.open || !h.m.active.ext.fromAlert {
		t.Fatalf("extend input not open from the alert")
	}
	h.typeText("10m")
	h.press("enter")
	tk := h.m.active.task
	if len(tk.Extensions) != 1 || tk.Extensions[0].Added != 13*time.Minute {
		t.Fatalf("extensions = %+v, want one of 13m", tk.Extensions)
	}
	if !strings.Contains(h.view(), "Remaining 10:00") {
		t.Errorf("view:\n%s", h.view())
	}
}
