// Package tui is unitock's terminal UI (Bubble Tea v2, D-04). The root Model routes
// messages to the current screen; screens never write to the store from View, and every
// state change is stored before the UI shows it (R-02).
package tui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/azntaiji/unitock/internal/alarm"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// Store is the persistence the UI needs. *store.Store satisfies it.
type Store interface {
	Create(ctx context.Context, project, name string, estimate time.Duration, startedAt time.Time) (task.Task, error)
	Projects(ctx context.Context) ([]string, error)
	Active(ctx context.Context) (task.Task, error)
	Complete(ctx context.Context, id int64, endedAt time.Time) error
	Cancel(ctx context.Context, id int64, endedAt time.Time) error
	Pause(ctx context.Context, id int64, at time.Time) (task.Task, error)
	AddExtension(ctx context.Context, id int64, at time.Time, added time.Duration) (task.Task, error)
	Resume(ctx context.Context, id int64, at time.Time) (task.Task, error)
	Completed(ctx context.Context, from, to time.Time) ([]task.Task, error)
	All(ctx context.Context) ([]task.Task, error)
	Setting(ctx context.Context, key string) (value string, ok bool, err error)
	SetSetting(ctx context.Context, key, value string) error
}

// Deps are the UI's collaborators.
type Deps struct {
	Ctx   context.Context
	Store Store
	Clock task.Clock
	Log   *slog.Logger
	// Rand returns a number in [0, n). Defaults to math/rand/v2.IntN.
	Rand func(n int) int
	// Alarm notifies and sounds at time up. Defaults to doing nothing.
	Alarm Alarm
	// Version is the app version written into exports (R-50).
	Version string
	// ExportDir is where exports are written. Empty means the working directory at the
	// time of the export (R-50).
	ExportDir string
}

type screen int

const (
	screenStart screen = iota
	screenActive
	screenDone
	screenStats
	screenSound
)

// Model is the root Bubble Tea model.
type Model struct {
	deps   Deps
	screen screen
	start  startModel
	active activeModel
	done   doneModel
	stats  statsModel
	snd    soundModel
	// sound is the saved alarm sound (R-53).
	sound alarm.Sound
	// statsFrom is the screen Stats was opened from; esc returns to it unchanged (R-46).
	statsFrom screen
	gen       int // tick loop generation; bumped for each Active screen
	width     int
	height    int
	theme     theme
}

// New returns the root model. If a task is active it opens on the Active screen with the
// timer where wall-clock time puts it (R-05); otherwise on the Start screen.
func New(d Deps) (Model, error) {
	if d.Ctx == nil {
		d.Ctx = context.Background()
	}
	if d.Log == nil {
		d.Log = slog.New(slog.DiscardHandler)
	}
	if d.Rand == nil {
		d.Rand = rand.IntN
	}
	if d.Alarm == nil {
		d.Alarm = nopAlarm{}
	}
	m := Model{deps: d, screen: screenStart, start: newStartModel(), theme: theme{}}
	m.sound = loadSound(d)
	d.Alarm.SetSound(m.sound)
	t, err := d.Store.Active(d.Ctx)
	switch {
	case errors.Is(err, store.ErrNoActive):
		return m, nil
	case err != nil:
		return Model{}, fmt.Errorf("loading active task: %w", err)
	}
	d.Log.Info("resuming task", "id", t.ID)
	m, _ = m.openActive(t)
	return m, nil
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.screen == screenActive {
		return m.active.tick()
	}
	return m.startCmds()
}

// startCmds are the Start screen's opening commands: cursor blink and loading the
// projects for suggestions (R-26).
func (m Model) startCmds() tea.Cmd {
	return tea.Batch(m.start.focusCmd(), loadProjects(m.deps))
}

// MinWidth and MinHeight are the smallest terminal the screens fit (R-08).
const (
	MinWidth  = 60
	MinHeight = 20
)

// tooSmall reports whether the terminal is below the minimum size. Before the first
// size message arrives the size is unknown, and the screen is shown.
func (m Model) tooSmall() bool {
	return m.width > 0 && m.height > 0 && (m.width < MinWidth || m.height < MinHeight)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.ColorProfileMsg: // NO_COLOR or no colour support (D-09)
		m.theme.noColor = msg.Profile <= colorprofile.ASCII
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" { // always quits (R-06), silencing any alarm (R-31)
			m.deps.Alarm.Stop()
			return m, tea.Quit
		}
		if m.tooSmall() { // the screen is hidden, so its keys are too (R-08)
			return m, nil
		}
		if msg.String() == "s" && m.canOpenStats() {
			return m.openStats()
		}
		if msg.String() == "a" && m.screen == screenStart && !m.start.inputFocused() { // R-51
			return m.openSound(), nil
		}
	case soundSavedMsg:
		return m.soundSaved(msg)
	case taskStartedMsg:
		m.deps.Log.Info("task started", "id", msg.task.ID)
		m.start = newStartModel()
		return m.openActive(msg.task)
	case taskCancelledMsg:
		m.deps.Log.Info("task cancelled", "id", msg.task.ID)
		m.screen = screenStart
		m.start = newStartModel()
		return m, m.startCmds()
	case taskCompletedMsg:
		m.deps.Log.Info("task completed", "id", msg.task.ID)
		m.screen = screenDone
		m.done = doneModel{task: msg.task, message: messages[m.deps.Rand(len(messages))]}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenStart:
		m.start, cmd = m.start.update(msg, m.deps)
	case screenActive:
		m.active, cmd = m.active.update(msg, m.deps)
	case screenStats:
		return m.updateStats(msg)
	case screenSound:
		return m.updateSound(msg)
	case screenDone:
		if k, ok := msg.(tea.KeyPressMsg); ok { // any key opens Start (R-23)
			if k.String() == "q" { // except q, which quits from any screen (R-06)
				return m, tea.Quit
			}
			m.screen = screenStart
			m.start = newStartModel()
			cmd = m.startCmds()
		}
	}
	return m, cmd
}

// canOpenStats reports whether `s` opens Stats now: from Start only with the button
// focused (R-14), and from Active only when no input, confirmation, or alert is open
// (R-24).
func (m Model) canOpenStats() bool {
	switch m.screen {
	case screenStart:
		return !m.start.inputFocused()
	case screenActive:
		return !m.active.ext.open && !m.active.conf.open && !m.active.alert.open
	}
	return false
}

// openStats shows the Stats screen on 7 days (R-40) and loads its data. The screen it
// was opened from keeps its state, and an active task's tick loop keeps running (R-24).
func (m Model) openStats() (Model, tea.Cmd) {
	m.statsFrom = m.screen
	m.screen = screenStats
	m.stats = statsModel{}
	return m, loadStats(m.deps)
}

// overActive reports whether Stats is open over an active task, whose timer and alert
// carry on underneath (R-24, R-30).
func (m Model) overActive() bool { return m.screen == screenStats && m.statsFrom == screenActive }

// updateStats handles a message while Stats is open. The active task's ticks and store
// results still go to it, so the alert can fire over Stats; while the alert is open it
// takes the keys (R-31). Choosing Extend switches to Active, where the duration input is.
func (m Model) updateStats(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg, taskUpdatedMsg, saveErrMsg:
		if m.overActive() {
			m.active, cmd = m.active.update(msg, m.deps)
		}
		return m, cmd
	case tea.KeyPressMsg:
		if m.overActive() && m.active.alert.open {
			m.active, cmd = m.active.update(msg, m.deps)
			if m.active.ext.open {
				m.screen = screenActive
			}
			return m, cmd
		}
		switch msg.String() {
		case "esc":
			m.screen = m.statsFrom
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}
	m.stats, cmd = m.stats.update(msg, m.deps)
	return m, cmd
}

// openActive shows t on the Active screen and starts its tick loop.
func (m Model) openActive(t task.Task) (Model, tea.Cmd) {
	m.gen++
	m.screen = screenActive
	m.active = newActiveModel(t, m.gen).checkTimeUp(m.deps.Clock.Now(), m.deps)
	return m, m.active.tick()
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var body, hints string
	switch {
	case m.tooSmall():
		body = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Terminal too small"),
			fmt.Sprintf("Resize to at least %d×%d (now %d×%d).", MinWidth, MinHeight, m.width, m.height),
		)
		hints = "ctrl+c quit"
	case m.screen == screenStart:
		body, hints = m.start.view(), m.start.hints()
	case m.screen == screenActive:
		body, hints = m.active.view(m.deps.Clock.Now(), m.width, m.theme), m.active.hints()
	case m.screen == screenDone:
		body, hints = m.done.view(), m.done.hints()
	case m.screen == screenSound:
		body, hints = m.soundView(), soundHints()
	case m.screen == screenStats:
		body, hints = m.stats.view(m.deps.Clock.Now(), m.width, m.height, m.theme), m.stats.hints()
		if m.overActive() && m.active.alert.open {
			body = overlay(body, m.active.alertView(m.theme, lipgloss.Width(body)))
			hints = m.active.hints()
		}
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, body, "", renderHints(hints)))
	v.AltScreen = true
	return v
}
