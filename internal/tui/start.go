package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/duration"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// MaxNameLen is the longest task name accepted, in characters (R-15).
const MaxNameLen = 100

type focus int

const (
	focusName focus = iota
	focusDuration
	focusButton
	focusCount
)

type taskStartedMsg struct{ task task.Task }

type startErrMsg struct{ err error }

// startModel is the Start screen (R-12 to R-16).
type startModel struct {
	name     textinput.Model
	duration textinput.Model
	focus    focus
	err      string
	starting bool // a create is in flight; ignore further enters
}

func newStartModel() startModel {
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "What are you working on?"
	name.SetWidth(50)

	dur := textinput.New()
	dur.Prompt = ""
	dur.Placeholder = "e.g. 25m, 1h30m, 1:30"
	dur.SetWidth(50)

	m := startModel{name: name, duration: dur}
	m.applyFocus()
	return m
}

func (m *startModel) applyFocus() {
	m.name.Blur()
	m.duration.Blur()
	switch m.focus {
	case focusName:
		m.name.Focus()
	case focusDuration:
		m.duration.Focus()
	}
}

func (m startModel) focusCmd() tea.Cmd {
	if m.focus == focusButton {
		return nil
	}
	return textinput.Blink
}

func (m startModel) inputFocused() bool { return m.focus != focusButton }

func (m startModel) moveFocus(delta int) (startModel, tea.Cmd) {
	m.focus = focus((int(m.focus) + delta + int(focusCount)) % int(focusCount))
	m.applyFocus()
	return m, m.focusCmd()
}

func (m startModel) update(msg tea.Msg, d Deps) (startModel, tea.Cmd) {
	switch msg := msg.(type) {
	case startErrMsg:
		m.starting = false
		if errors.Is(msg.err, store.ErrActiveExists) {
			m.err = "Another task is already active."
		} else {
			d.Log.Error("create task", "err", msg.err)
			m.err = "Could not save the task: " + msg.err.Error()
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down":
			return m.moveFocus(1)
		case "shift+tab", "up":
			return m.moveFocus(-1)
		case "enter":
			return m.submit(d)
		case "esc":
			m.focus = focusButton
			m.applyFocus()
			return m, nil
		}
		if !m.inputFocused() {
			if msg.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
	}

	// Everything else (typing, cursor blink) goes to the focused input.
	var cmd tea.Cmd
	switch m.focus {
	case focusName:
		m.name, cmd = m.name.Update(msg)
	case focusDuration:
		m.duration, cmd = m.duration.Update(msg)
	}
	return m, cmd
}

// validate checks the inputs (R-15). It returns the trimmed name and parsed duration,
// or a message to show inline.
func (m startModel) validate() (name string, dur time.Duration, problem string) {
	name = strings.TrimSpace(m.name.Value())
	if name == "" {
		return "", 0, "Task name is required."
	}
	if n := utf8.RuneCountInString(name); n > MaxNameLen {
		return "", 0, fmt.Sprintf("Task name is %d characters; the limit is %d.", n, MaxNameLen)
	}
	dur, err := duration.Parse(m.duration.Value())
	if err != nil {
		return "", 0, "Duration: " + err.Error()
	}
	return name, dur, ""
}

func (m startModel) submit(d Deps) (startModel, tea.Cmd) {
	if m.starting {
		return m, nil
	}
	name, dur, problem := m.validate()
	if problem != "" {
		m.err = problem
		return m, nil
	}
	m.err = ""
	m.starting = true
	return m, func() tea.Msg {
		t, err := d.Store.Create(d.Ctx, name, dur, d.Clock.Now())
		if err != nil {
			return startErrMsg{err}
		}
		return taskStartedMsg{t}
	}
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	labelStyle   = lipgloss.NewStyle().Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Red)
	buttonStyle  = lipgloss.NewStyle().Padding(0, 2).Border(lipgloss.RoundedBorder())
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Blue)
)

func (m startModel) view() string {
	label := func(s string, f focus) string {
		if m.focus == f {
			return focusedStyle.Render("> " + s)
		}
		return labelStyle.Render("  " + s)
	}
	button := buttonStyle
	if m.focus == focusButton {
		button = button.Reverse(true).BorderForeground(lipgloss.Blue)
	}
	lines := []string{
		titleStyle.Render("unitock"),
		"",
		label("Task name", focusName),
		"  " + m.name.View(),
		"",
		label("Task duration", focusDuration),
		"  " + m.duration.View(),
		"",
		button.MarginLeft(2).Render("Start task"),
	}
	if m.err != "" {
		lines = append(lines, "", errStyle.Render("  "+m.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// hints lists the keys that work in the current focus (R-07, R-14).
func (m startModel) hints() string {
	if m.inputFocused() {
		return "tab next · ↑/↓ move · enter start · esc button · ctrl+c quit"
	}
	return "tab next · ↑/↓ move · enter start · q quit"
}
