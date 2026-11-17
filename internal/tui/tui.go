// Package tui is unitock's terminal UI (Bubble Tea v2, D-04). The root Model routes
// messages to the current screen; screens never write to the store from View, and every
// state change is stored before the UI shows it (R-02).
package tui

import (
	"context"
	"log/slog"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/task"
)

// Store is the persistence the UI needs. *store.Store satisfies it.
type Store interface {
	Create(ctx context.Context, name string, estimate time.Duration, startedAt time.Time) (task.Task, error)
	Active(ctx context.Context) (task.Task, error)
	Complete(ctx context.Context, id int64, endedAt time.Time) error
}

// Deps are the UI's collaborators.
type Deps struct {
	Ctx   context.Context
	Store Store
	Clock task.Clock
	Log   *slog.Logger
}

type screen int

const (
	screenStart screen = iota
	screenActive
)

// Model is the root Bubble Tea model.
type Model struct {
	deps   Deps
	screen screen
	start  startModel
	active activeModel
	width  int
	height int
}

// New returns the root model, opening on the Start screen.
func New(d Deps) Model {
	if d.Ctx == nil {
		d.Ctx = context.Background()
	}
	if d.Log == nil {
		d.Log = slog.New(slog.DiscardHandler)
	}
	return Model{deps: d, screen: screenStart, start: newStartModel()}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return m.start.focusCmd() }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" { // always quits (R-06)
			return m, tea.Quit
		}
	case taskStartedMsg:
		m.deps.Log.Info("task started", "id", msg.task.ID)
		m.screen = screenActive
		m.active = newActiveModel(msg.task)
		m.start = newStartModel()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenStart:
		m.start, cmd = m.start.update(msg, m.deps)
	case screenActive:
		m.active, cmd = m.active.update(msg)
	}
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var body, hints string
	switch m.screen {
	case screenStart:
		body, hints = m.start.view(), m.start.hints()
	case screenActive:
		body, hints = m.active.view(), m.active.hints()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, body, "", renderHints(hints)))
	v.AltScreen = true
	return v
}
