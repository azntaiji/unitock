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

// taskUpdatedMsg carries the active task after a stored change, such as a pause.
type taskUpdatedMsg struct{ task task.Task }

// saveErrMsg reports a failed store write from the Active screen.
type saveErrMsg struct {
	op  string
	err error
}

// activeModel is the Active screen.
type activeModel struct {
	task task.Task
	gen  int
	busy bool // a store write is in flight; further writes wait for it
	err  string
}

func newActiveModel(t task.Task, gen int) activeModel { return activeModel{task: t, gen: gen} }

// tick schedules the next redraw on the system clock's next whole second. Unlike
// tea.Tick, whose interval starts only after the previous tick is handled, this does not
// drift, so the countdown never skips a second (R-19).
func (m activeModel) tick() tea.Cmd {
	gen := m.gen
	return tea.Every(tickInterval, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

func (m activeModel) update(msg tea.Msg, d Deps) (activeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		return m, m.tick()
	case taskUpdatedMsg:
		m.busy = false
		m.task = msg.task
		return m, nil
	case saveErrMsg:
		m.busy = false
		d.Log.Error(msg.op+" task", "id", m.task.ID, "err", msg.err)
		m.err = "Could not save: " + msg.err.Error()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "p":
			return m.save(d, func(t task.Task, now time.Time) tea.Msg {
				op, fn := "pause", d.Store.Pause
				if t.Paused() {
					op, fn = "resume", d.Store.Resume
				}
				t, err := fn(d.Ctx, t.ID, now)
				if err != nil {
					return saveErrMsg{op, err}
				}
				return taskUpdatedMsg{t}
			})
		case "e":
			return m.save(d, func(t task.Task, now time.Time) tea.Msg {
				if err := d.Store.Complete(d.Ctx, t.ID, now); err != nil {
					return saveErrMsg{"complete", err}
				}
				t.Status, t.EndedAt = task.Completed, now
				return taskCompletedMsg{t}
			})
		}
	}
	return m, nil
}

// save runs one store write for the task at the clock's current time, unless another
// is in flight. The UI changes only when the write's result message arrives (R-02).
func (m activeModel) save(d Deps, write func(t task.Task, now time.Time) tea.Msg) (activeModel, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	m.busy = true
	m.err = ""
	t := m.task
	return m, func() tea.Msg { return write(t, d.Clock.Now()) }
}

// view draws the Active screen (R-17, R-18, D-09): name, status badge, timer in block
// digits (plain bold when narrower than blockMinWidth), progress bar, and the remaining
// and estimate as text. Paused dims the digits.
func (m activeModel) view(now time.Time, width int, th theme) string {
	b := bandAt(m.task, now)
	fg := b.color(th)
	digits := lipgloss.NewStyle().Bold(true).Foreground(fg).Faint(m.task.Paused())
	clock := timerText(m.task, now)
	timer := digits.Render(clock)
	if width == 0 || width >= blockMinWidth {
		timer = digits.Render(blockText(clock))
	}
	left := "Remaining " + clock
	if b == bandOvertime {
		left = "Overtime " + clock
	}
	lines := append([]string{badge(m.task, b, th)}, heading(m.task)...)
	lines = append(lines,
		"",
		timer,
		"",
		progressBar(m.task, now, fg),
		"",
		dimStyle.Render(left+" · Estimate "+formatClock(m.task.Estimate())),
	)
	if m.err != "" {
		lines = append(lines, "", errStyle.Render(m.err))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m activeModel) hints() string {
	if m.task.Paused() {
		return "p resume · e complete · q quit"
	}
	return "p pause · e complete · q quit"
}

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
