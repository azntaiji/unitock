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

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// Store is the persistence the UI needs. *store.Store satisfies it.
type Store interface {
	Create(ctx context.Context, project, name string, estimate time.Duration, startedAt time.Time) (task.Task, error)
	Projects(ctx context.Context) ([]string, error)
	Active(ctx context.Context) (task.Task, error)
	Complete(ctx context.Context, id int64, endedAt time.Time) error
	Pause(ctx context.Context, id int64, at time.Time) (task.Task, error)
	AddExtension(ctx context.Context, id int64, at time.Time, added time.Duration) (task.Task, error)
	Resume(ctx context.Context, id int64, at time.Time) (task.Task, error)
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
	theme  theme
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
	m := Model{deps: d, screen: screenStart, start: newStartModel(), theme: theme{}}
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
		body, hints = m.active.view(m.deps.Clock.Now(), m.width, m.theme), m.active.hints()
	case screenDone:
		body, hints = m.done.view(), m.done.hints()
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, body, "", renderHints(hints)))
	v.AltScreen = true
	return v
}
