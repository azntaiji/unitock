package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	bkey "charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/duration"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// MaxNameLen is the longest task name accepted, in characters (R-15).
const MaxNameLen = 100

// MaxProjectLen is the longest project name accepted, in characters (R-25).
const MaxProjectLen = 50

// maxSuggestions is how many projects the dropdown lists (R-26).
const maxSuggestions = 4

type focus int

const (
	focusProject focus = iota
	focusName
	focusDuration
	focusButton
	focusCount
)

type taskStartedMsg struct{ task task.Task }

type startErrMsg struct{ err error }

// projectsMsg carries the stored project names, most recently used first.
type projectsMsg struct {
	names []string
	err   error
}

// startModel is the Start screen (R-12 to R-16).
type startModel struct {
	project  textinput.Model
	projects []string // every stored project, most recently used first
	pick     int      // highlighted suggestion, or -1 for none
	closed   bool     // esc closed the dropdown; typing reopens it
	name     textinput.Model
	duration textinput.Model
	focus    focus
	err      string
	starting bool // a create is in flight; ignore further enters
}

func newStartModel() startModel {
	project := textinput.New()
	project.Prompt = ""
	project.Placeholder = "Optional; tab to skip"
	project.SetWidth(50)
	// The input only draws the ghost text; the Start screen owns the keys (R-26).
	project.ShowSuggestions = true
	project.KeyMap.AcceptSuggestion = bkey.NewBinding(bkey.WithDisabled())
	project.KeyMap.NextSuggestion = bkey.NewBinding(bkey.WithDisabled())
	project.KeyMap.PrevSuggestion = bkey.NewBinding(bkey.WithDisabled())

	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "What are you working on?"
	name.SetWidth(50)

	dur := textinput.New()
	dur.Prompt = ""
	dur.Placeholder = "e.g. 25m, 1h30m, 1:30"
	dur.SetWidth(50)

	m := startModel{project: project, pick: -1, name: name, duration: dur}
	m.applyFocus()
	return m
}

func (m *startModel) applyFocus() {
	m.project.Blur()
	m.name.Blur()
	m.duration.Blur()
	m.pick, m.closed = -1, false
	switch m.focus {
	case focusProject:
		m.project.Focus()
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

// loadProjects reads the stored projects for the suggestions.
func loadProjects(d Deps) tea.Cmd {
	return func() tea.Msg {
		names, err := d.Store.Projects(d.Ctx)
		return projectsMsg{names, err}
	}
}

// suggestions are the projects the dropdown lists for the current input (R-26): the
// most recent when it is empty, otherwise those containing it ignoring case, prefix
// matches first, each group in most-recent order.
func (m startModel) suggestions() []string {
	q := strings.ToLower(strings.TrimSpace(m.project.Value()))
	var prefix, other []string
	for _, p := range m.projects {
		lp := strings.ToLower(p)
		switch {
		case strings.HasPrefix(lp, q):
			prefix = append(prefix, p)
		case strings.Contains(lp, q):
			other = append(other, p)
		}
	}
	out := append(prefix, other...)
	return out[:min(len(out), maxSuggestions)]
}

// dropdownOpen reports whether the project suggestions are showing.
func (m startModel) dropdownOpen() bool {
	return m.focus == focusProject && !m.closed && len(m.suggestions()) > 0
}

// completion is the suggestion tab would fill in: the highlighted one, or the top one
// once something is typed. Empty when there is none or it equals the input.
func (m startModel) completion() string {
	if !m.dropdownOpen() {
		return ""
	}
	sug := m.suggestions()
	c := ""
	switch {
	case m.pick >= 0 && m.pick < len(sug):
		c = sug[m.pick]
	case strings.TrimSpace(m.project.Value()) != "":
		c = sug[0]
	}
	if c == m.project.Value() {
		return ""
	}
	return c
}

// refreshGhost shows the completion as ghost text when it extends the typed text.
func (m *startModel) refreshGhost() {
	c := m.completion()
	if c != "" && m.project.Value() != "" && strings.HasPrefix(strings.ToLower(c), strings.ToLower(m.project.Value())) {
		m.project.SetSuggestions([]string{c})
		return
	}
	m.project.SetSuggestions(nil)
}

// fillProject puts p in the project input with the cursor at the end.
func (m *startModel) fillProject(p string) {
	m.project.SetValue(p)
	m.project.CursorEnd()
	m.pick = -1
	m.refreshGhost()
}

// dropdownKey handles a key while the project suggestions are open (R-13, R-26).
// handled is false for keys the dropdown does not use.
func (m startModel) dropdownKey(k string, d Deps) (_ startModel, _ tea.Cmd, handled bool) {
	n := len(m.suggestions())
	switch k {
	case "tab":
		if c := m.completion(); c != "" {
			m.fillProject(c)
			return m, nil, true
		}
		m, cmd := m.moveFocus(1)
		return m, cmd, true
	case "down":
		m.pick = min(m.pick+1, n-1)
	case "up":
		m.pick = max(m.pick-1, -1)
	case "enter":
		if m.pick < 0 {
			m, cmd := m.submit(d)
			return m, cmd, true
		}
		m.fillProject(m.suggestions()[m.pick])
		return m, nil, true
	case "esc":
		m.closed, m.pick = true, -1
	default:
		return m, nil, false
	}
	m.refreshGhost()
	return m, nil, true
}

func (m startModel) moveFocus(delta int) (startModel, tea.Cmd) {
	m.focus = focus((int(m.focus) + delta + int(focusCount)) % int(focusCount))
	m.applyFocus()
	return m, m.focusCmd()
}

func (m startModel) update(msg tea.Msg, d Deps) (startModel, tea.Cmd) {
	switch msg := msg.(type) {
	case projectsMsg:
		if msg.err != nil {
			d.Log.Error("list projects", "err", msg.err)
		}
		m.projects = msg.names
		m.refreshGhost()
		return m, nil
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
		if m.dropdownOpen() {
			if next, cmd, ok := m.dropdownKey(msg.String(), d); ok {
				return next, cmd
			}
		}
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
			return m, nil
		}
	}

	// Everything else (typing, cursor blink) goes to the focused input.
	var cmd tea.Cmd
	switch m.focus {
	case focusProject:
		before := m.project.Value()
		m.project, cmd = m.project.Update(msg)
		if m.project.Value() != before { // typing reopens the dropdown (R-26)
			m.closed, m.pick = false, -1
		}
		m.refreshGhost()
	case focusName:
		m.name, cmd = m.name.Update(msg)
	case focusDuration:
		m.duration, cmd = m.duration.Update(msg)
	}
	return m, cmd
}

// validate checks the inputs (R-15, R-25). It returns the trimmed project and name and
// the parsed duration, or a message to show inline.
func (m startModel) validate() (project, name string, dur time.Duration, problem string) {
	project = strings.TrimSpace(m.project.Value())
	if n := utf8.RuneCountInString(project); n > MaxProjectLen {
		return "", "", 0, fmt.Sprintf("Project is %d characters; the limit is %d.", n, MaxProjectLen)
	}
	name = strings.TrimSpace(m.name.Value())
	if name == "" {
		return "", "", 0, "Task name is required."
	}
	if n := utf8.RuneCountInString(name); n > MaxNameLen {
		return "", "", 0, fmt.Sprintf("Task name is %d characters; the limit is %d.", n, MaxNameLen)
	}
	dur, err := duration.Parse(m.duration.Value())
	if err != nil {
		return "", "", 0, "Duration: " + err.Error()
	}
	return project, name, dur, ""
}

func (m startModel) submit(d Deps) (startModel, tea.Cmd) {
	if m.starting {
		return m, nil
	}
	project, name, dur, problem := m.validate()
	if problem != "" {
		m.err = problem
		return m, nil
	}
	m.err = ""
	m.starting = true
	return m, func() tea.Msg {
		t, err := d.Store.Create(d.Ctx, project, name, dur, d.Clock.Now())
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
	pickStyle    = lipgloss.NewStyle().Reverse(true)
	dimStyle     = lipgloss.NewStyle().Faint(true)
)

// dropdownView lists the project suggestions under the input, the highlighted one in
// reverse video (R-26).
func (m startModel) dropdownView() []string {
	if !m.dropdownOpen() {
		return nil
	}
	var out []string
	for i, p := range m.suggestions() {
		if i == m.pick {
			out = append(out, "  "+pickStyle.Render(" "+p+" "))
		} else {
			out = append(out, "  "+dimStyle.Render(" "+p))
		}
	}
	return out
}

func (m startModel) view(alarmLabel string) string {
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
		label("Project", focusProject),
		"  " + m.project.View(),
	}
	lines = append(lines, m.dropdownView()...)
	lines = append(lines,
		"",
		label("Task name", focusName),
		"  "+m.name.View(),
		"",
		label("Task duration", focusDuration),
		"  "+m.duration.View(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Center, button.MarginLeft(2).Render("Start task"), "  "+dimStyle.Render("Alarm: "+alarmLabel)),
	)
	if m.err != "" {
		lines = append(lines, "", errText(2, m.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// hints lists the keys that work in the current focus (R-07, R-14).
func (m startModel) hints() string {
	if m.dropdownOpen() {
		enter := "enter start"
		if m.pick >= 0 {
			enter = "enter choose"
		}
		// One line, so the screen still fits with the dropdown open (R-08).
		return "tab complete · " + enter + " · esc close · ctrl+c quit"
	}
	if m.inputFocused() {
		return "tab next · enter start · esc button\n" + chordHints("ctrl+s stats", "ctrl+o sound")
	}
	return "tab next · enter start\n" + chordHints("ctrl+s stats", "ctrl+o sound")
}
