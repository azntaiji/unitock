package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/alarm"
	"github.com/azntaiji/unitock/internal/store"
)

// soundModel is the Sound screen (R-51 to R-53, D-16): the alarm sounds, with the saved
// one marked and the highlighted one previewed.
type soundModel struct {
	cursor int    // index into alarm.Sounds
	saving bool   // a save is in flight
	err    string // the last save's error
}

// soundSavedMsg reports a save of the alarm sound.
type soundSavedMsg struct {
	sound alarm.Sound
	err   error
}

// loadSound reads the saved alarm sound. A missing, unknown, or unreadable value means
// the default (R-53); the last two are logged.
func loadSound(d Deps) alarm.Sound {
	v, ok, err := d.Store.Setting(d.Ctx, store.SettingAlarmSound)
	switch {
	case err != nil:
		d.Log.Warn("reading the alarm sound; using the default", "err", err)
		return alarm.DefaultSound
	case !ok:
		return alarm.DefaultSound
	}
	s, ok := alarm.ParseSound(v)
	if !ok {
		d.Log.Warn("unknown alarm sound; using the default", "stored", v)
		return alarm.DefaultSound
	}
	return s
}

// saveSound stores s as the alarm sound (R-53).
func saveSound(d Deps, s alarm.Sound) tea.Cmd {
	return func() tea.Msg {
		return soundSavedMsg{sound: s, err: d.Store.SetSetting(d.Ctx, store.SettingAlarmSound, string(s))}
	}
}

// openSound shows the Sound screen with the saved sound highlighted (R-51).
func (m Model) openSound() Model {
	m.screen = screenSound
	m.snd = soundModel{}
	for i, s := range alarm.Sounds {
		if s == m.sound {
			m.snd.cursor = i
		}
	}
	return m
}

// closeSound stops any preview and returns to Start (R-52).
func (m Model) closeSound() Model {
	m.deps.Alarm.Stop()
	m.screen = screenStart
	return m
}

// updateSound handles keys on the Sound screen (R-52). Moving the highlight previews
// the sound; enter saves it; esc leaves it unchanged.
func (m Model) updateSound(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "up", "down":
		next := m.snd.cursor - 1
		if k.String() == "down" {
			next = m.snd.cursor + 1
		}
		if next < 0 || next >= len(alarm.Sounds) {
			return m, nil
		}
		m.snd.cursor = next
		m.deps.Alarm.Preview(alarm.Sounds[next])
	case "enter":
		if m.snd.saving {
			return m, nil
		}
		m.snd.saving, m.snd.err = true, ""
		return m, saveSound(m.deps, alarm.Sounds[m.snd.cursor])
	case "esc":
		return m.closeSound(), nil
	}
	return m, nil
}

// soundSaved applies a finished save: the alarm uses the new sound and Start opens, or
// the error shows and the old choice stands (R-53).
func (m Model) soundSaved(msg soundSavedMsg) (tea.Model, tea.Cmd) {
	m.snd.saving = false
	if msg.err != nil {
		m.deps.Log.Error("saving the alarm sound", "err", msg.err)
		m.snd.err = "Could not save the sound: " + msg.err.Error()
		return m, nil
	}
	m.deps.Log.Info("alarm sound set", "sound", msg.sound)
	m.sound = msg.sound
	m.deps.Alarm.SetSound(msg.sound)
	if m.screen != screenSound { // left (esc) before the save finished
		return m, nil
	}
	return m.closeSound(), nil
}

func (m Model) soundView() string {
	lines := []string{titleStyle.Render("Alarm sound"), dimStyle.Render("Plays when a task's time is up."), ""}
	for i, s := range alarm.Sounds {
		label := s.Label()
		if s == m.sound {
			label += dimStyle.Render("  (current)")
		}
		if i == m.snd.cursor {
			lines = append(lines, focusedStyle.Render("> "+s.Label())+strings.TrimPrefix(label, s.Label()))
		} else {
			lines = append(lines, "  "+label)
		}
	}
	if m.snd.err != "" {
		lines = append(lines, "", errText(0, m.snd.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func soundHints() string { return "↑/↓ choose and play · enter save · esc back · ctrl+c quit" }
