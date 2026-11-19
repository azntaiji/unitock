package tui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/unitock/internal/alarm"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

type harness struct {
	t     *testing.T
	m     Model
	store *store.Store
	clock *task.FakeClock
	alarm *fakeAlarm
	quit  bool
}

// fakeAlarm records Alarm calls; playing is whether the alarm sound is on, previewing
// whether a preview is, and sound the sound set for the next alarm.
type fakeAlarm struct {
	notified   []string
	starts     int
	playing    bool
	previews   []alarm.Sound
	previewing bool
	sound      alarm.Sound
	stops      int
}

func (a *fakeAlarm) Notify(name string)     { a.notified = append(a.notified, name) }
func (a *fakeAlarm) Start()                 { a.starts++; a.playing, a.previewing = true, false }
func (a *fakeAlarm) Stop()                  { a.stops++; a.playing, a.previewing = false, false }
func (a *fakeAlarm) SetSound(s alarm.Sound) { a.sound = s }
func (a *fakeAlarm) Preview(s alarm.Sound) {
	if !a.playing {
		a.previews, a.previewing = append(a.previews, s), true
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := task.NewFakeClock(t0)
	al := &fakeAlarm{}
	m, err := New(Deps{Store: st, Clock: clk, Alarm: al})
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, m: m, store: st, clock: clk, alarm: al}
	h.run(loadProjects(m.deps)) // what Init does for the Start screen
	return h
}

// key builds a key press from a Bubble Tea key name ("tab", "shift+tab", "esc", ...)
// or a single printable character.
func key(s string) tea.KeyPressMsg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	}
	if l, ok := strings.CutPrefix(s, "ctrl+"); ok && len(l) == 1 {
		return tea.KeyPressMsg{Code: rune(l[0]), Mod: tea.ModCtrl}
	}
	r := []rune(s)
	if len(r) != 1 {
		panic("key: unknown key " + s)
	}
	return tea.KeyPressMsg{Code: r[0], Text: s}
}

// send delivers msg. For action keys (enter submits, e completes, p pauses, y confirms,
// q and ctrl+c quit) it runs the resulting command synchronously and feeds its message
// back in, the way the Bubble Tea runtime would. Other commands, such as cursor blinks and the Active
// screen's tick, wait on a timer and are dropped; tests send tickMsg themselves.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	next, cmd := h.m.Update(msg)
	h.m = next.(Model)
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "enter", "e", "p", "y", "ctrl+c", "ctrl+s":
			h.run(cmd)
		}
	}
}

func (h *harness) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		h.quit = true
	case taskStartedMsg, startErrMsg, taskCompletedMsg, taskUpdatedMsg, saveErrMsg, projectsMsg, exportedMsg, soundSavedMsg:
		h.send(msg)
	case taskCancelledMsg: // its follow-up (blink, project load) returns at once
		next, cmd := h.m.Update(msg)
		h.m = next.(Model)
		h.run(cmd)
	case tea.BatchMsg:
		for _, c := range msg {
			h.run(c)
		}
	}
}

func (h *harness) press(keys ...string) {
	h.t.Helper()
	for _, k := range keys {
		h.send(key(k))
	}
}

// typeText types s. Commands from typing are dropped, but a quit is recorded so tests
// can assert that typing never quits.
func (h *harness) typeText(s string) {
	h.t.Helper()
	for _, r := range s {
		next, cmd := h.m.Update(key(string(r)))
		h.m = next.(Model)
		if isQuit(cmd) {
			h.quit = true
		}
	}
}

// isQuit reports whether cmd is tea.Quit, without running it.
func isQuit(cmd tea.Cmd) bool {
	return cmd != nil && reflect.ValueOf(cmd).Pointer() == reflect.ValueOf(tea.Quit).Pointer()
}

func (h *harness) view() string { return h.m.View().Content }

func (h *harness) taskCount() int {
	h.t.Helper()
	_, err := h.store.Active(context.Background())
	if errors.Is(err, store.ErrNoActive) {
		return 0
	}
	if err != nil {
		h.t.Fatal(err)
	}
	return 1
}

func TestStartKeyNames(t *testing.T) {
	for _, k := range []string{"tab", "shift+tab", "enter", "esc", "up", "down", "left", "right", "ctrl+c", "ctrl+s", "ctrl+o", "q"} {
		if got := key(k).String(); got != k {
			t.Errorf("key(%q).String() = %q", k, got)
		}
	}
}

func TestStartFocusCycle(t *testing.T) {
	h := newHarness(t)
	want := func(f focus) {
		t.Helper()
		if h.m.start.focus != f {
			t.Fatalf("focus = %d, want %d", h.m.start.focus, f)
		}
		if h.m.start.project.Focused() != (f == focusProject) || h.m.start.name.Focused() != (f == focusName) ||
			h.m.start.duration.Focused() != (f == focusDuration) {
			t.Fatalf("input focus out of sync with focus %d", f)
		}
	}
	want(focusProject) // no projects stored: no dropdown, so keys move focus
	h.press("tab")
	want(focusName)
	h.press("tab")
	want(focusDuration)
	h.press("tab")
	want(focusButton)
	h.press("tab")
	want(focusProject)
	h.press("down", "down", "down")
	want(focusButton)
	h.press("up")
	want(focusDuration)
	h.press("shift+tab", "up")
	want(focusProject)
}

func TestStartTypingQAndSDoesNotQuitOrLeave(t *testing.T) {
	h := newHarness(t)
	h.typeText("qs")
	if h.quit || h.m.start.project.Value() != "qs" {
		t.Fatalf("project field: quit=%v value=%q", h.quit, h.m.start.project.Value())
	}
	h.press("tab")
	h.typeText("quick sums")
	if h.quit {
		t.Fatal("typing q in the name field quit")
	}
	if h.m.screen != screenStart {
		t.Fatal("typing left the Start screen")
	}
	if got := h.m.start.name.Value(); got != "quick sums" {
		t.Errorf("name = %q, want %q", got, "quick sums")
	}
	h.press("tab")
	h.typeText("qs")
	if h.quit || h.m.start.duration.Value() != "qs" {
		t.Errorf("duration field: quit=%v value=%q", h.quit, h.m.start.duration.Value())
	}
}

// R-06: on the button, plain q and s do nothing; only chords are commands.
func TestStartEscFocusesButtonAndQDoesNotQuit(t *testing.T) {
	h := newHarness(t)
	h.typeText("x")
	h.press("esc")
	if h.m.start.focus != focusButton {
		t.Fatalf("esc: focus = %d, want button", h.m.start.focus)
	}
	h.press("q", "s", "a")
	if h.quit || h.m.screen != screenStart || h.m.start.focus != focusButton {
		t.Errorf("q, s, a on the button: quit %v screen %d focus %d; want nothing", h.quit, h.m.screen, h.m.start.focus)
	}
}

func TestStartCtrlCAlwaysQuits(t *testing.T) {
	h := newHarness(t)
	h.press("ctrl+c")
	if !h.quit {
		t.Error("ctrl+c in the name field did not quit")
	}
}

func TestStartHintsChangeWithFocus(t *testing.T) {
	h := newHarness(t)
	h.press("tab") // Task name: no dropdown
	inputHints := h.m.start.hints()
	for _, want := range []string{"esc button", "\nctrl+s stats · ctrl+o sound · ctrl+c quit"} {
		if !strings.Contains(inputHints, want) {
			t.Errorf("input hints = %q, want %q", inputHints, want)
		}
	}
	if strings.Contains(inputHints, "q quit") && !strings.Contains(inputHints, "ctrl+c quit") || strings.Contains(inputHints, "↑/↓") {
		t.Errorf("input hints = %q", inputHints)
	}
	for _, line := range strings.Split(inputHints, "\n") {
		if !strings.Contains(h.view(), line) {
			t.Errorf("view does not show hint line %q", line)
		}
	}
	h.press("esc")
	buttonHints := h.m.start.hints()
	if buttonHints == inputHints || !strings.Contains(buttonHints, "ctrl+s stats · ctrl+o sound · ctrl+c quit") {
		t.Errorf("button hints = %q", buttonHints)
	}
	for _, line := range strings.Split(buttonHints, "\n") {
		if !strings.Contains(h.view(), line) {
			t.Errorf("view does not show button hint line %q", line)
		}
	}
}

func TestStartValidation(t *testing.T) {
	tests := []struct {
		name, dur, want string
	}{
		{"", "25m", "Task name is required."},
		{"   ", "25m", "Task name is required."},
		{strings.Repeat("é", MaxNameLen+1), "25m", "limit is 100"},
		{"write", "", "1h30m"},
		{"write", "soon", "1h30m"},
		{"write", "30s", "from 1 minute to 24 hours"},
		{"write", "25h", "from 1 minute to 24 hours"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.dur, func(t *testing.T) {
			h := newHarness(t)
			h.press("tab")
			h.typeText(tt.name)
			h.press("tab")
			h.typeText(tt.dur)
			h.press("enter")
			if !strings.Contains(h.m.start.err, tt.want) {
				t.Errorf("err = %q, want it to contain %q", h.m.start.err, tt.want)
			}
			if !strings.Contains(h.view(), tt.want) {
				t.Error("error not shown in the view")
			}
			if h.m.screen != screenStart || h.taskCount() != 0 {
				t.Error("invalid input started a task")
			}
		})
	}
}

func TestStartCreatesTaskAndOpensActive(t *testing.T) {
	h := newHarness(t)
	name := "  " + strings.Repeat("n", MaxNameLen) + "  "
	h.press("tab")
	h.typeText(name)
	h.press("tab")
	h.typeText("1h 30m")
	h.press("shift+tab") // enter works from any focus
	h.press("enter")

	if h.m.screen != screenActive {
		t.Fatalf("screen = %d, want Active; err %q", h.m.screen, h.m.start.err)
	}
	got, err := h.store.Active(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != strings.TrimSpace(name) || got.OriginalEstimate != 90*time.Minute || !got.StartedAt.Equal(t0) {
		t.Errorf("stored %+v", got)
	}
	if !strings.Contains(h.view(), got.Name) {
		t.Error("Active screen does not show the task name")
	}
}

func TestStartFromButton(t *testing.T) {
	h := newHarness(t)
	h.press("tab")
	h.typeText("plan")
	h.press("tab")
	h.typeText("25")
	h.press("tab", "enter")
	if h.m.screen != screenActive || h.taskCount() != 1 {
		t.Fatalf("enter on the button did not start: err %q", h.m.start.err)
	}
}

func TestStartWithActiveTaskShowsError(t *testing.T) {
	h := newHarness(t)
	if _, err := h.store.Create(context.Background(), "", "other", time.Hour, t0); err != nil {
		t.Fatal(err)
	}
	h.press("tab")
	h.typeText("second")
	h.press("tab")
	h.typeText("10m")
	h.press("enter")
	if h.m.screen != screenStart || !strings.Contains(h.m.start.err, "already active") {
		t.Errorf("screen %d err %q", h.m.screen, h.m.start.err)
	}
	if h.m.start.starting {
		t.Error("still marked as starting after the error")
	}
}

func TestStartIsQuitDetectsQuit(t *testing.T) {
	if !isQuit(tea.Quit) || isQuit(nil) || isQuit(func() tea.Msg { return nil }) {
		t.Fatal("isQuit misclassifies commands")
	}
	h := newHarness(t)
	h.press("esc")
	h.typeText("q") // not a quit any more (R-06)
	if h.quit {
		t.Fatal("q quit")
	}
	next, cmd := h.m.Update(key("ctrl+c"))
	h.m = next.(Model)
	if !isQuit(cmd) {
		t.Fatal("ctrl+c did not return a quit")
	}
}
