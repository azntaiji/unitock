package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/task"
)

// messages are shown at random when a task is completed (R-23).
var messages = []string{
	"Nice work. One thing done, fully.",
	"That's a finished task. Enjoy the feeling.",
	"Done is better than perfect, and this is done.",
	"Another one off the list. Keep the streak going.",
	"Focus paid off. Well done.",
	"You showed up and finished. That's the whole trick.",
	"Momentum is yours. Ride it or rest; both are earned.",
	"Small steps, finished steps. Great job.",
	"Task complete. Take a breath before the next one.",
	"You said you'd do it, and you did.",
	"Progress, not perfection. Nicely done.",
	"That's how big things get built: one task at a time.",
	"Finished. Your future self says thanks.",
	"Clear head, clear list. Good going.",
	"One less thing on your mind.",
	"Steady and done. Excellent.",
	"You stayed with it to the end. Respect.",
	"Another block of real work, banked.",
	"Good finish. Stretch, sip some water, carry on.",
	"Done and dusted.",
	"That task never stood a chance.",
	"Well played. On to whatever's next.",
}

// doneModel is the message shown after completing a task; any key opens Start.
type doneModel struct {
	task    task.Task
	message string
}

var messageStyle = lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(true)

func (m doneModel) view() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Completed: "+m.task.Name),
		"",
		messageStyle.Render(m.message),
	)
}

func (m doneModel) hints() string { return "any key continue · q quit" }
