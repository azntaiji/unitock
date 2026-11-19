package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/duration"
	"github.com/azntaiji/unitock/internal/task"
)

// tickInterval is how often the Active screen redraws (R-19: at least once a second).
const tickInterval = time.Second

// tickMsg asks the Active screen to redraw. gen ties it to one tick loop, so a loop
// left over from an earlier task stops instead of doubling the tick rate.
type tickMsg struct{ gen int }

type taskCompletedMsg struct{ task task.Task }

type taskCancelledMsg struct{ task task.Task }

// taskUpdatedMsg carries the active task after a stored change, such as a pause.
type taskUpdatedMsg struct{ task task.Task }

// saveErrMsg reports a failed store write from the Active screen.
type saveErrMsg struct {
	op  string
	err error
}

// activeModel is the Active screen.
type activeModel struct {
	task  task.Task
	gen   int
	busy  bool // a store write is in flight; further writes wait for it
	err   string
	ext   extendInput
	conf  confirmCancel
	alert timeUpAlert
}

// confirmCancel is the cancel confirmation that `n` opens (R-22). No is selected first,
// so a stray enter does not cancel.
type confirmCancel struct {
	open bool
	yes  bool // Yes is selected
}

// extendInput is the duration input that `d` opens to add time (R-21).
type extendInput struct {
	open      bool
	fromAlert bool // opened from the time-up alert: D-10 counts the overtime too
	input     textinput.Model
	err       string
}

func newActiveModel(t task.Task, gen int) activeModel { return activeModel{task: t, gen: gen} }

// openExtend shows the duration input. It works while running or paused (R-21).
func (m activeModel) openExtend(fromAlert bool) (activeModel, tea.Cmd) {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "e.g. 10m, 1h"
	in.SetWidth(30)
	m.ext = extendInput{open: true, fromAlert: fromAlert, input: in}
	m.err = ""
	return m, m.ext.input.Focus()
}

// extendKey handles a key while the duration input is open: enter adds the amount,
// esc closes unchanged, anything else types (so q, p, e, and d are text here, R-06).
func (m activeModel) extendKey(msg tea.KeyPressMsg, d Deps) (activeModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.ext.fromAlert {
			return m.backToAlert(d), nil
		}
		m.ext = extendInput{}
		return m, nil
	case "enter":
		entered, err := duration.Parse(m.ext.input.Value())
		if err != nil {
			m.ext.err = "Duration: " + err.Error()
			return m, nil
		}
		m.ext.err = ""
		fromAlert := m.ext.fromAlert
		return m.save(d, func(t task.Task, now time.Time) tea.Msg {
			t, err := d.Store.AddExtension(d.Ctx, t.ID, now, t.ExtensionFor(now, entered, fromAlert))
			if err != nil {
				return saveErrMsg{"extend", err}
			}
			return taskUpdatedMsg{t}
		})
	}
	var cmd tea.Cmd
	m.ext.input, cmd = m.ext.input.Update(msg)
	return m, cmd
}

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
		m = m.checkTimeUp(d.Clock.Now(), d)
		return m, m.tick()
	case taskUpdatedMsg:
		m.busy = false
		m.task = msg.task
		m.ext = extendInput{}               // a stored extension closes the input
		m = m.checkTimeUp(d.Clock.Now(), d) // e.g. resumed in overtime
		return m, nil
	case saveErrMsg:
		m.busy = false
		m.conf = confirmCancel{}
		d.Log.Error(msg.op+" task", "id", m.task.ID, "err", msg.err)
		m.err = "Could not save: " + msg.err.Error()
		return m, nil
	case tea.KeyPressMsg:
		if m.ext.open {
			return m.extendKey(msg, d)
		}
		if m.alert.open {
			return m.alertKey(msg, d)
		}
		if m.conf.open {
			return m.confirmKey(msg, d)
		}
		switch msg.String() {
		case "d":
			return m.openExtend(false)
		case "n":
			m.conf = confirmCancel{open: true}
			m.err = ""
			return m, nil
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
			return m.complete(d)
		}
		return m, nil
	}
	if m.ext.open { // cursor blink and other input messages
		var cmd tea.Cmd
		m.ext.input, cmd = m.ext.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// confirmKey handles a key while the cancel confirmation is open (R-22): y, or enter on
// Yes, cancels the task; n, esc, or enter on No closes it unchanged; ←/→/tab choose.
// q still quits (R-06); other Active keys are inactive.
func (m activeModel) confirmKey(msg tea.KeyPressMsg, d Deps) (activeModel, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "tab", "shift+tab":
		m.conf.yes = !m.conf.yes
	case "n", "esc":
		m.conf = confirmCancel{}
	case "enter":
		if !m.conf.yes {
			m.conf = confirmCancel{}
			return m, nil
		}
		return m.cancel(d)
	case "y":
		return m.cancel(d)
	}
	return m, nil
}

// complete stores the task as completed; the root model then shows the Done screen.
func (m activeModel) complete(d Deps) (activeModel, tea.Cmd) {
	return m.save(d, func(t task.Task, now time.Time) tea.Msg {
		if err := d.Store.Complete(d.Ctx, t.ID, now); err != nil {
			return saveErrMsg{"complete", err}
		}
		t.Status, t.EndedAt = task.Completed, now
		return taskCompletedMsg{t}
	})
}

// cancel stores the task as cancelled; the root model then opens the Start screen.
func (m activeModel) cancel(d Deps) (activeModel, tea.Cmd) {
	return m.save(d, func(t task.Task, now time.Time) tea.Msg {
		if err := d.Store.Cancel(d.Ctx, t.ID, now); err != nil {
			return saveErrMsg{"cancel", err}
		}
		t.Status, t.EndedAt = task.Cancelled, now
		return taskCancelledMsg{t}
	})
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

// overlay draws box on top of screen, centred both ways, so the alert floats over the
// screen beneath it (R-30). The result is as wide and tall as the larger of the two.
func overlay(screen, box string) string {
	x := max(0, (lipgloss.Width(screen)-lipgloss.Width(box))/2)
	y := max(0, (lipgloss.Height(screen)-lipgloss.Height(box))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(screen),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	).Render()
}

// view draws the Active screen (R-17, R-18, D-09): name, status badge, timer in block
// digits (plain bold when narrower than blockMinWidth), progress bar, and the elapsed
// time and estimate as text. Paused dims the digits.
func (m activeModel) view(now time.Time, width int, th theme) string {
	b := bandAt(m.task, now)
	fg := b.color(th)
	digits := lipgloss.NewStyle().Bold(true).Foreground(fg).Faint(m.task.Paused())
	clock := timerText(m.task, now)
	timer := digits.Render(clock)
	if width == 0 || width >= blockMinWidth {
		timer = digits.Render(blockText(clock))
	}
	lines := append([]string{badge(m.task, b, th)}, heading(m.task)...)
	lines = append(lines,
		"",
		timer,
		"",
		progressBar(m.task, now, fg),
		"",
		dimStyle.Render(timeRow(m.task, now)),
	)
	if m.ext.open {
		lines = append(lines, "", focusedStyle.Render("> Add time"), "  "+m.ext.input.View())
		if m.ext.err != "" {
			lines = append(lines, errStyle.Render("  "+m.ext.err))
		}
	}
	if m.conf.open {
		yes, no := "  Yes  ", "  No  "
		if m.conf.yes {
			yes = pickStyle.Render(yes)
		} else {
			no = pickStyle.Render(no)
		}
		lines = append(lines, "", errStyle.Render("Cancel this task?")+"  "+yes+" "+no)
	}
	if m.err != "" {
		lines = append(lines, "", errStyle.Render(m.err))
	}
	screen := lipgloss.JoinVertical(lipgloss.Left, lines...)
	if m.alert.open {
		return overlay(screen, m.alertView(th, lipgloss.Width(screen)))
	}
	return screen
}

func (m activeModel) hints() string {
	if m.ext.open && m.ext.fromAlert {
		return "enter add · esc back · ctrl+c quit"
	}
	if m.ext.open {
		return "enter add · esc cancel · ctrl+c quit"
	}
	if m.alert.open {
		return "←/→ choose · enter select · d extend · e complete · ctrl+c quit"
	}
	if m.conf.open {
		return "y yes · n/esc no · ←/→ choose · enter select · ctrl+c quit"
	}
	p := "p pause"
	if m.task.Paused() {
		p = "p resume"
	}
	return p + " · d add time · n cancel · e complete\n" + chordHints("ctrl+s stats")
}

// timeRow is the line under the progress bar (R-17): elapsed time and the original
// estimate, plus the total added once the task has been extended. The big digits
// already show the remaining time, so it is not repeated. Elapsed rounds down while
// the digits round up, so the two always sum to the estimate.
func timeRow(t task.Task, now time.Time) string {
	row := "Elapsed " + formatClock(t.Elapsed(now)) + " · Estimate " + formatClock(t.OriginalEstimate)
	if added := t.Estimate() - t.OriginalEstimate; added > 0 {
		row += " + " + formatClock(added) + " added"
	}
	return row
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
