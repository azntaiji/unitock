package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/task"
)

// tickInterval is how often the Active screen redraws (R-19: at least once a second).
const tickInterval = time.Second

// tickMsg asks the Active screen to redraw. gen ties it to one tick loop, so a loop
// left over from an earlier task stops instead of doubling the tick rate.
type tickMsg struct{ gen int }

type taskCompletedMsg struct{ task task.Task }

type completeErrMsg struct{ err error }

// activeModel is the Active screen. Phase 1 shows the name and a countdown; pause,
// extend, cancel, and the big digits come in Phase 2.
type activeModel struct {
	task       task.Task
	gen        int
	completing bool
	err        string
}

func newActiveModel(t task.Task, gen int) activeModel { return activeModel{task: t, gen: gen} }

func (m activeModel) tick() tea.Cmd {
	gen := m.gen
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

func (m activeModel) update(msg tea.Msg, d Deps) (activeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		return m, m.tick()
	case completeErrMsg:
		m.completing = false
		d.Log.Error("complete task", "id", m.task.ID, "err", msg.err)
		m.err = "Could not save: " + msg.err.Error()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "e":
			if m.completing {
				return m, nil
			}
			m.completing = true
			m.err = ""
			t := m.task
			return m, func() tea.Msg {
				end := d.Clock.Now()
				if err := d.Store.Complete(d.Ctx, t.ID, end); err != nil {
					return completeErrMsg{err}
				}
				t.Status, t.EndedAt = task.Completed, end
				return taskCompletedMsg{t}
			}
		}
	}
	return m, nil
}

var timerStyle = lipgloss.NewStyle().Bold(true)

func (m activeModel) view(now time.Time) string {
	lines := []string{
		titleStyle.Render(m.task.Name),
		"",
		timerStyle.Render(timerText(m.task, now)),
		fmt.Sprintf("Estimate %s", formatClock(m.task.Estimate())),
	}
	if m.err != "" {
		lines = append(lines, "", errStyle.Render(m.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m activeModel) hints() string { return "e complete · q quit" }

// timerText is the remaining time, or +overtime once the estimate is used up (R-17, R-18).
// Remaining rounds up so the display reads the full estimate at the start and 00:00
// exactly at time up; overtime rounds down.
func timerText(t task.Task, now time.Time) string {
	if ot := t.Overtime(now); ot > 0 {
		return "+" + formatClock(ot.Truncate(time.Second))
	}
	r := t.Remaining(now)
	if rem := r % time.Second; rem != 0 {
		r += time.Second - rem
	}
	return formatClock(r)
}

// formatClock renders d as H:MM:SS, or MM:SS under an hour (R-17). Sub-second parts
// are dropped.
func formatClock(d time.Duration) string {
	s := int64(d / time.Second)
	h, m, sec := s/3600, s/60%60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
