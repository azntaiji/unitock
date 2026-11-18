package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// openCancel starts a task and presses n, checking the confirmation shows.
func (h *harness) openCancel() task.Task {
	h.t.Helper()
	tk := h.startTask("write", "25m")
	h.clock.Advance(2 * time.Minute)
	h.press("n")
	if !h.m.active.conf.open || !strings.Contains(h.view(), "Cancel this task?") || !strings.Contains(h.view(), "y yes") {
		h.t.Fatalf("n did not open the confirmation:\n%s", h.view())
	}
	return tk
}

func (h *harness) wantCancelled(id int64) {
	h.t.Helper()
	if h.m.screen != screenStart {
		h.t.Fatalf("screen = %d, want Start", h.m.screen)
	}
	got, err := h.store.Get(context.Background(), id)
	if err != nil {
		h.t.Fatal(err)
	}
	if got.Status != task.Cancelled || !got.EndedAt.Equal(t0.Add(2*time.Minute)) {
		h.t.Errorf("stored %+v, want cancelled at +2m", got)
	}
	if _, err := h.store.Active(context.Background()); !errors.Is(err, store.ErrNoActive) {
		h.t.Errorf("Active after cancel: %v", err)
	}
}

// P2-04: y cancels and shows the Start screen (R-22).
func TestCancelYConfirms(t *testing.T) {
	h := newHarness(t)
	tk := h.openCancel()
	h.press("y")
	h.wantCancelled(tk.ID)
}

func TestCancelEnterOnYesConfirms(t *testing.T) {
	h := newHarness(t)
	tk := h.openCancel()
	h.press("left", "enter")
	h.wantCancelled(tk.ID)
}

// n, esc, and enter on the default No leave the task running (R-22).
func TestCancelNoLeavesTaskRunning(t *testing.T) {
	for _, keys := range [][]string{{"n"}, {"esc"}, {"enter"}, {"tab", "tab", "enter"}} {
		t.Run(strings.Join(keys, "+"), func(t *testing.T) {
			h := newHarness(t)
			tk := h.openCancel()
			h.press(keys...)
			if h.m.screen != screenActive || h.m.active.conf.open {
				t.Fatalf("screen %d open %v", h.m.screen, h.m.active.conf.open)
			}
			got, err := h.store.Get(context.Background(), tk.ID)
			if err != nil || got.Status != task.Active {
				t.Errorf("stored %+v, %v; want active", got, err)
			}
			if !strings.Contains(h.view(), "n cancel") {
				t.Errorf("hints after closing:\n%s", h.view())
			}
		})
	}
}

// While confirming, other Active keys do nothing; q still quits (R-06).
func TestCancelOtherKeysInactive(t *testing.T) {
	h := newHarness(t)
	h.openCancel()
	h.press("p", "e", "d")
	if h.m.active.task.Paused() || h.m.screen != screenActive || h.m.active.ext.open || !h.m.active.conf.open {
		t.Fatalf("keys acted during confirmation: paused=%v screen=%d ext=%v", h.m.active.task.Paused(), h.m.screen, h.m.active.ext.open)
	}
	h.press("q")
	if !h.quit {
		t.Error("q did not quit from the confirmation")
	}
}

// After a cancel the Start screen reloads projects, so a new project is suggested.
func TestCancelReloadsProjects(t *testing.T) {
	h := newHarness(t)
	h.typeText("Side quest")
	h.press("tab")
	h.typeText("write")
	h.press("tab")
	h.typeText("25m")
	h.press("enter", "n", "y")
	if h.m.screen != screenStart || len(h.m.start.suggestions()) != 1 {
		t.Errorf("screen %d suggestions %v", h.m.screen, h.m.start.suggestions())
	}
}

func TestCancelErrorStays(t *testing.T) {
	h := newHarness(t)
	h.openCancel()
	_ = h.store.Close()
	h.press("y")
	if h.m.screen != screenActive || !strings.Contains(h.m.active.err, "Could not save") || h.m.active.busy {
		t.Errorf("screen %d err %q busy %v", h.m.screen, h.m.active.err, h.m.active.busy)
	}
}
