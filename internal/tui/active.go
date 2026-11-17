package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/unitock/internal/task"
)

// activeModel is the Active screen. Placeholder: the countdown, `e`, and messages come
// in the next Phase 1 task.
type activeModel struct {
	task task.Task
}

func newActiveModel(t task.Task) activeModel { return activeModel{task: t} }

func (m activeModel) update(msg tea.Msg) (activeModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}

func (m activeModel) view() string { return titleStyle.Render(m.task.Name) }

func (m activeModel) hints() string { return "q quit" }
