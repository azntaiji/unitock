package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// startTask drives the Start screen to begin a task.
func (h *harness) startTask(name, dur string) task.Task {
	h.t.Helper()
	h.typeText(name)
	h.press("tab")
	h.typeText(dur)
	h.press("enter")
	if h.m.screen != screenActive {
		h.t.Fatalf("task did not start: %q", h.m.start.err)
	}
	return h.m.active.task
}

func (h *harness) tick() { h.t.Helper(); h.send(tickMsg{gen: h.m.gen}) }

func TestFormatClock(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00"},
		{59 * time.Second, "00:59"},
		{25 * time.Minute, "25:00"},
		{59*time.Minute + 59*time.Second + 999*time.Millisecond, "59:59"},
		{time.Hour, "1:00:00"},
		{24 * time.Hour, "24:00:00"},
		{90*time.Minute + 5*time.Second, "1:30:05"},
	}
	for _, tt := range tests {
		if got := formatClock(tt.d); got != tt.want {
			t.Errorf("formatClock(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestActiveTimerText(t *testing.T) {
	tk := task.Task{StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	tests := []struct {
		after time.Duration
		want  string
	}{
		{0, "25:00"},
		{500 * time.Millisecond, "25:00"}, // remaining rounds up
		{time.Second, "24:59"},
		{24*time.Minute + 59*time.Second + 1, "00:01"},
		{25 * time.Minute, "00:00"},
		{25*time.Minute + 500*time.Millisecond, "+00:00"}, // overtime rounds down
		{26*time.Minute + time.Second, "+01:01"},
	}
	for _, tt := range tests {
		if got := timerText(tk, t0.Add(tt.after)); got != tt.want {
			t.Errorf("after %v: %q, want %q", tt.after, got, tt.want)
		}
	}
}

func TestActiveCountdownTicks(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	if v := h.view(); !strings.Contains(v, "write") || !strings.Contains(v, "25:00") {
		t.Fatalf("Active view = %q", v)
	}
	for i, want := range []string{"24:59", "24:58", "24:57"} {
		h.clock.Advance(time.Second)
		h.tick()
		if !strings.Contains(h.view(), want) {
			t.Fatalf("tick %d: view lacks %q:\n%s", i+1, want, h.view())
		}
	}
	h.clock.Advance(10 * time.Minute)
	h.tick()
	if !strings.Contains(h.view(), "14:57") {
		t.Errorf("after 10m view = %q", h.view())
	}
}

func TestActiveTickLoopIsGenerational(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	_, cmd := h.m.active.update(tickMsg{gen: h.m.gen}, h.m.deps)
	if cmd == nil {
		t.Error("current tick did not schedule the next")
	}
	_, cmd = h.m.active.update(tickMsg{gen: h.m.gen - 1}, h.m.deps)
	if cmd != nil {
		t.Error("stale tick scheduled another")
	}
}

func TestActiveCompleteShowsMessageThenStart(t *testing.T) {
	h := newHarness(t)
	h.m.deps.Rand = func(n int) int { return n - 1 }
	tk := h.startTask("write", "25m")
	h.clock.Advance(20 * time.Minute)
	h.press("e")

	if h.m.screen != screenDone {
		t.Fatalf("screen = %d, want Done; err %q", h.m.screen, h.m.active.err)
	}
	if v := h.view(); !strings.Contains(v, messages[len(messages)-1]) {
		t.Errorf("Done view lacks the chosen message:\n%s", v)
	}
	got, err := h.store.Get(context.Background(), tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != task.Completed || !got.EndedAt.Equal(t0.Add(20*time.Minute)) {
		t.Errorf("stored status %q ended %v", got.Status, got.EndedAt)
	}

	h.press("x") // any key
	if h.m.screen != screenStart || h.m.start.name.Value() != "" {
		t.Fatalf("any key: screen = %d", h.m.screen)
	}
	h.startTask("next", "5m") // the completed task no longer blocks a new one
}

func TestActiveQuitLeavesTaskActive(t *testing.T) {
	h := newHarness(t)
	tk := h.startTask("write", "25m")
	h.press("q")
	if !h.quit {
		t.Fatal("q did not quit")
	}
	got, err := h.store.Active(context.Background())
	if err != nil || got.ID != tk.ID {
		t.Errorf("after quit Active = %+v, %v", got, err)
	}
}

func TestActiveCompleteErrorStays(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	_ = h.store.Close()
	h.press("e")
	if h.m.screen != screenActive || !strings.Contains(h.m.active.err, "Could not save") {
		t.Errorf("screen %d err %q", h.m.screen, h.m.active.err)
	}
	if h.m.active.completing {
		t.Error("still completing after the error")
	}
}

func TestActiveHints(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	for _, k := range []string{"e complete", "q quit"} {
		if !strings.Contains(h.view(), k) {
			t.Errorf("hints lack %q", k)
		}
	}
}

func TestMessagesDistinct(t *testing.T) {
	if len(messages) < 20 {
		t.Fatalf("%d messages, want at least 20 (R-23)", len(messages))
	}
	seen := map[string]bool{}
	for _, m := range messages {
		if strings.TrimSpace(m) == "" || seen[m] {
			t.Errorf("empty or duplicate message %q", m)
		}
		seen[m] = true
	}
}

// Compile-time check that the real store satisfies the UI's interface.
var _ Store = (*store.Store)(nil)
