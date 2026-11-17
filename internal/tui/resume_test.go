package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// relaunch opens a fresh store and model on dir, as a new process would.
func relaunch(t *testing.T, dir string, clk task.Clock) (Model, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	m, err := New(Deps{Store: st, Clock: clk})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m, st
}

func TestResumeOpensActiveWithClosedTimeElapsed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	clk := task.NewFakeClock(t0)

	// First run: start a 25-minute task, then quit.
	m, st := relaunch(t, dir, clk)
	h := &harness{t: t, m: m, store: st, clock: clk}
	tk := h.startTask("write", "25m")
	clk.Advance(2 * time.Minute)
	h.press("q")
	if !h.quit {
		t.Fatal("q did not quit")
	}
	_ = st.Close()

	// The app stays closed for 10m20s, then relaunches.
	clk.Advance(10*time.Minute + 20*time.Second)
	m, _ = relaunch(t, dir, clk)

	if m.screen != screenActive || m.active.task.ID != tk.ID {
		t.Fatalf("relaunch screen = %d task %d, want Active task %d", m.screen, m.active.task.ID, tk.ID)
	}
	if v := m.View().Content; !strings.Contains(v, "12:40") || !strings.Contains(v, "write") {
		t.Errorf("resumed view lacks 12:40:\n%s", v)
	}
	if m.Init() == nil {
		t.Error("resumed Active screen did not start its tick loop")
	}
}

func TestResumeWithoutActiveTaskOpensStart(t *testing.T) {
	dir := t.TempDir()
	clk := task.NewFakeClock(t0)
	m, st := relaunch(t, dir, clk)
	if m.screen != screenStart {
		t.Fatalf("empty store: screen = %d, want Start", m.screen)
	}

	// A completed task is not resumed.
	tk, err := st.Create(context.Background(), "old", time.Hour, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Complete(context.Background(), tk.ID, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	m, _ = relaunch(t, dir, clk)
	if m.screen != screenStart {
		t.Errorf("after completion: screen = %d, want Start", m.screen)
	}
}

func TestResumeStoreErrorIsReturned(t *testing.T) {
	st, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	if _, err := New(Deps{Store: st, Clock: task.NewFakeClock(t0)}); err == nil {
		t.Error("New with a closed store succeeded")
	}
}

func TestDoneScreenKeys(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.press("e")
	if !strings.Contains(h.view(), "q quit") || !strings.Contains(h.view(), "any key") {
		t.Errorf("Done hints:\n%s", h.view())
	}
	h.press("q")
	if !h.quit || h.m.screen != screenDone {
		t.Errorf("q on Done: quit=%v screen=%d", h.quit, h.m.screen)
	}

	h = newHarness(t)
	h.startTask("write", "25m")
	h.press("e", "enter")
	if h.quit || h.m.screen != screenStart {
		t.Errorf("enter on Done: quit=%v screen=%d", h.quit, h.m.screen)
	}
}

func TestEveryScreenShowsHints(t *testing.T) {
	h := newHarness(t)
	check := func(name string) {
		t.Helper()
		if !strings.Contains(h.view(), "quit") {
			t.Errorf("%s screen has no quit hint:\n%s", name, h.view())
		}
	}
	check("Start")
	h.startTask("write", "25m")
	check("Active")
	h.press("e")
	check("Done")
}
