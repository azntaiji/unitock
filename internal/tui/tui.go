// Package tui is unitock's terminal UI (Bubble Tea v2, D-04). The root Model routes
// messages to the current screen; screens never write to the store from View, and every
// state change is stored before the UI shows it (R-02).
package tui

import (
	"context"
	"log/slog"
	"math/rand/v2"
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
	// Rand returns a number in [0, n). Defaults to math/rand/v2.IntN.
	Rand func(n int) int
}

type screen int

const (
	screenStart screen = iota
	screenActive
	screenDone
)

// Model is the root Bubble Tea model.
type Model struct {
	deps   Deps
	screen screen
	start  startModel
	active activeModel
	done   doneModel
	gen    int // tick loop generation; bumped for each Active screen
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
	if d.Rand == nil {
		d.Rand = rand.IntN
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
		m.start = newStartModel()
		return m.openActive(msg.task)
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
	case screenDone:
		if _, ok := msg.(tea.KeyPressMsg); ok { // any key opens Start (R-23)
			m.screen = screenStart
			m.start = newStartModel()
			cmd = m.start.focusCmd()
		}
	}
	return m, cmd
}

// openActive shows t on the Active screen and starts its tick loop.
func (m Model) openActive(t task.Task) (Model, tea.Cmd) {
	m.gen++
	m.screen = screenActive
	m.active = newActiveModel(t, m.gen)
	return m, m.active.tick()
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var body, hints string
	switch m.screen {
	case screenStart:
		body, hints = m.start.view(), m.start.hints()
	case screenActive:
		body, hints = m.active.view(m.deps.Clock.Now()), m.active.hints()
	case screenDone:
		body, hints = m.done.view(), m.done.hints()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, body, "", renderHints(hints)))
	v.AltScreen = true
	return v
}
