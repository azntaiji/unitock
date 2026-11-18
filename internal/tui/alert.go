package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Alarm is the system side of the time-up alert (R-30, R-33, R-34). Every method must
// return at once: implementations run tools in the background and log failures, never
// returning them (D-13). Start and Stop are idempotent.
type Alarm interface {
	// Notify sends one system notification that taskName's time is up.
	Notify(taskName string)
	// Start begins the looping alarm sound.
	Start()
	// Stop ends the alarm sound.
	Stop()
}

type nopAlarm struct{}

func (nopAlarm) Notify(string) {}
func (nopAlarm) Start()        {}
func (nopAlarm) Stop()         {}

// timeUpAlert is the modal shown when a running task reaches zero (R-30, R-31).
type timeUpAlert struct {
	open     bool
	complete bool // Mark as completed is selected; Extend is selected first
	fired    bool // the alert has fired for the current time-up; reset once time is added
}

// checkTimeUp opens the alert, sends the notification, and starts the sound the first
// time a running task is at or past its estimate (R-30). It also fires at once for a
// task that ran out while the app was closed (R-05). A paused task never fires.
func (m activeModel) checkTimeUp(now time.Time, d Deps) activeModel {
	if !m.task.TimeUp(now) {
		m.alert.fired = false
		return m
	}
	if m.alert.fired || m.task.Paused() {
		return m
	}
	m.alert = timeUpAlert{open: true, fired: true}
	m.ext, m.conf = extendInput{}, confirmCancel{}
	m.err = ""
	d.Alarm.Notify(m.task.Name)
	d.Alarm.Start()
	return m
}

// alertKey handles a key while the alert is open (R-31): ←/→/tab choose, enter selects,
// d and e select directly, q quits. Other keys are inactive. Choosing either option
// stops the sound.
func (m activeModel) alertKey(msg tea.KeyPressMsg, d Deps) (activeModel, tea.Cmd) {
	switch k := msg.String(); k {
	case "left", "right", "tab", "shift+tab":
		m.alert.complete = !m.alert.complete
	case "enter", "d", "e":
		if k == "e" || (k == "enter" && m.alert.complete) {
			d.Alarm.Stop()
			m.alert.open = false
			return m.complete(d)
		}
		d.Alarm.Stop()
		m.alert.open = false
		return m.openExtend(true) // D-10: remaining becomes the amount entered (R-32)
	case "q":
		d.Alarm.Stop()
		return m, tea.Quit
	}
	return m, nil
}

// backToAlert reopens the alert when its Extend input is escaped (R-32). The sound
// resumes, since the alert is unanswered; the notification is not sent again.
func (m activeModel) backToAlert(d Deps) activeModel {
	m.ext = extendInput{}
	m.alert.open = true
	d.Alarm.Start()
	return m
}

var alertBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 2)

// alertView is the modal box: a heading, the task, and the two choices (R-31). It is at
// least minWidth wide, so it can cover the whole screen width beneath it.
func (m activeModel) alertView(th theme, minWidth int) string {
	extend, complete := "  Extend  ", "  Mark as completed  "
	if m.alert.complete {
		complete = pickStyle.Render(complete)
	} else {
		extend = pickStyle.Render(extend)
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(bandOvertime.color(th)).Render("▲ Time's up")
	body := lipgloss.JoinVertical(lipgloss.Left, title, m.task.Name, "", extend+" "+complete)
	// Width counts the padding and border.
	w := max(lipgloss.Width(body)+alertBoxStyle.GetHorizontalFrameSize(), minWidth)
	return alertBoxStyle.Width(w).BorderForeground(bandOvertime.color(th)).Render(body)
}
