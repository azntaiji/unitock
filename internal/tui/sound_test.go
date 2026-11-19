package tui

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/unitock/internal/alarm"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// openSoundScreen presses ctrl+o on Start.
func (h *harness) openSoundScreen() {
	h.t.Helper()
	h.press("ctrl+o")
	if h.m.screen != screenSound {
		h.t.Fatalf("ctrl+o did not open the Sound screen (screen %d)", h.m.screen)
	}
}

func (h *harness) savedSound() (string, bool) {
	h.t.Helper()
	v, ok, err := h.store.Setting(context.Background(), store.SettingAlarmSound)
	if err != nil {
		h.t.Fatal(err)
	}
	return v, ok
}

// R-51, R-14: ctrl+o opens the Sound screen from any Start focus, even while typing;
// plain a types in an input and does nothing on the button. Start shows the saved
// sound beside the button.
func TestSoundScreenOpensFromStartAnyFocus(t *testing.T) {
	h := newHarness(t)
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Start task  │  Alarm: Chime") {
		t.Errorf("Start lacks Alarm: Chime beside the button:\n%s", v)
	}
	h.press("esc", "a")
	if h.m.screen != screenStart {
		t.Fatalf("plain a on the button: screen %d, want Start", h.m.screen)
	}
	h.press("tab", "tab") // Task name
	h.typeText("a")
	if h.m.screen != screenStart || h.m.start.name.Value() != "a" {
		t.Fatalf("a in the name input: screen %d, name %q", h.m.screen, h.m.start.name.Value())
	}
	if !strings.Contains(h.view(), "ctrl+o sound") {
		t.Errorf("Start hints while typing lack ctrl+o sound:\n%s", h.view())
	}
	h.press("ctrl+o")
	if h.m.screen != screenSound {
		t.Fatalf("ctrl+o while typing: screen %d, want Sound", h.m.screen)
	}
	v := ansi.Strip(h.view())
	for _, want := range []string{"Alarm sound", "Chime  (current)", "Bell", "Beep", "Marimba", "Rising", "↑/↓ choose and play", "ctrl+c quit"} {
		if !strings.Contains(v, want) {
			t.Errorf("Sound screen lacks %q:\n%s", want, v)
		}
	}
	if h.m.snd.cursor != 0 || len(h.alarm.previews) != 0 {
		t.Errorf("opened with cursor %d, previews %v; want the saved sound highlighted, silent", h.m.snd.cursor, h.alarm.previews)
	}
}

// R-52: moving the highlight previews each sound once; the ends do not wrap.
func TestSoundScreenPreviewsOnMove(t *testing.T) {
	h := newHarness(t)
	h.openSoundScreen()
	h.press("up") // already at the top
	h.press("down", "down", "up")
	for range 6 { // past the bottom
		h.press("down")
	}
	want := []alarm.Sound{alarm.Bell, alarm.Beep, alarm.Bell, alarm.Beep, alarm.Marimba, alarm.Rising}
	if !reflect.DeepEqual(h.alarm.previews, want) {
		t.Errorf("previews %v, want %v", h.alarm.previews, want)
	}
	if h.m.snd.cursor != len(alarm.Sounds)-1 {
		t.Errorf("cursor %d, want the last sound", h.m.snd.cursor)
	}
}

// R-52, R-53: enter saves the highlighted sound, the alarm uses it, the preview stops,
// and the choice survives a relaunch.
func TestSoundScreenEnterSavesAndAlarmUsesIt(t *testing.T) {
	h := newHarness(t)
	h.openSoundScreen()
	h.press("down") // Bell
	h.press("enter")
	if h.m.screen != screenStart {
		t.Fatalf("after enter: screen %d, want Start", h.m.screen)
	}
	if v, ok := h.savedSound(); !ok || v != "bell" {
		t.Errorf("stored %q, %v; want bell", v, ok)
	}
	if h.alarm.sound != alarm.Bell || h.alarm.previewing {
		t.Errorf("alarm sound %q, previewing %v; want bell, stopped", h.alarm.sound, h.alarm.previewing)
	}
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Alarm: Bell") {
		t.Errorf("Start does not show the new sound:\n%s", v)
	}
	h.press("ctrl+o")
	if h.m.snd.cursor != 1 || !strings.Contains(ansi.Strip(h.view()), "Bell  (current)") {
		t.Errorf("reopened: cursor %d\n%s", h.m.snd.cursor, ansi.Strip(h.view()))
	}

	relaunched := &fakeAlarm{}
	m, err := New(Deps{Store: h.store, Clock: h.clock, Alarm: relaunched})
	if err != nil {
		t.Fatal(err)
	}
	if relaunched.sound != alarm.Bell || m.sound != alarm.Bell {
		t.Errorf("after relaunch the alarm sound is %q, model %q; want bell", relaunched.sound, m.sound)
	}
}

// R-52, R-06: esc returns without changing the choice and stops the preview; plain q
// does nothing and ctrl+c quits.
func TestSoundScreenEscLeavesChoiceAndCtrlCQuits(t *testing.T) {
	h := newHarness(t)
	h.openSoundScreen()
	h.press("down", "down") // Beep, previewing
	h.press("esc")
	if h.m.screen != screenStart || h.m.sound != alarm.Chime || h.alarm.previewing {
		t.Errorf("after esc: screen %d, sound %q, previewing %v", h.m.screen, h.m.sound, h.alarm.previewing)
	}
	if v, ok := h.savedSound(); ok {
		t.Errorf("esc stored %q", v)
	}
	h.press("ctrl+o", "down", "q")
	if h.quit || h.m.screen != screenSound {
		t.Errorf("plain q on Sound: quit %v, screen %d", h.quit, h.m.screen)
	}
	h.press("ctrl+c")
	if !h.quit || h.alarm.previewing {
		t.Errorf("ctrl+c: quit %v, previewing %v", h.quit, h.alarm.previewing)
	}
}

// R-53: a missing or unknown stored value means Chime.
func TestSoundLoadFallsBackToChime(t *testing.T) {
	for _, stored := range []string{"", "siren", "Bell"} {
		h := newHarness(t)
		if stored != "" {
			if err := h.store.SetSetting(context.Background(), store.SettingAlarmSound, stored); err != nil {
				t.Fatal(err)
			}
		}
		al := &fakeAlarm{}
		m, err := New(Deps{Store: h.store, Clock: h.clock, Alarm: al})
		if err != nil {
			t.Fatal(err)
		}
		if al.sound != alarm.Chime || m.sound != alarm.Chime {
			t.Errorf("stored %q: alarm %q, model %q; want chime", stored, al.sound, m.sound)
		}
	}
}

// settingErrStore is a Store whose settings reads and writes fail.
type settingErrStore struct{ Store }

func (settingErrStore) Setting(context.Context, string) (string, bool, error) {
	return "", false, errDisk
}
func (settingErrStore) SetSetting(context.Context, string, string) error { return errDisk }

// R-53: a failed save shows the error and keeps the old choice; a failed read at
// startup means Chime, not a failed start.
func TestSoundSaveErrorShownAndChoiceKept(t *testing.T) {
	h := newHarness(t)
	h.m.deps.Store = settingErrStore{h.m.deps.Store}
	h.openSoundScreen()
	h.press("down", "enter")
	if h.m.screen != screenSound || h.m.sound != alarm.Chime || h.alarm.sound != alarm.Chime {
		t.Errorf("after a failed save: screen %d, sound %q, alarm %q", h.m.screen, h.m.sound, h.alarm.sound)
	}
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Could not save the sound: disk on fire") {
		t.Errorf("error not shown:\n%s", v)
	}

	al := &fakeAlarm{}
	m, err := New(Deps{Store: settingErrStore{h.store}, Clock: task.NewFakeClock(t0), Alarm: al})
	if err != nil || m.sound != alarm.Chime || al.sound != alarm.Chime {
		t.Errorf("unreadable setting at startup: %v, model %q, alarm %q", err, m.sound, al.sound)
	}
}

func TestSoundView(t *testing.T) {
	h := newHarness(t)
	h.openSoundScreen()
	h.press("down", "down") // Beep highlighted, Chime current
	requireGolden(t, "sound", h.m.soundView())
}
