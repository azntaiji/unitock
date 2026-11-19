package tui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/azntaiji/unitock/internal/export"
	"github.com/azntaiji/unitock/internal/stats"
	"github.com/azntaiji/unitock/internal/task"
)

// statsLoadedMsg carries the completed tasks the Stats screen summarises, or the error
// reading them.
type statsLoadedMsg struct {
	tasks []task.Task
	err   error
}

// loadStats reads every completed task any period could need: from the start of the
// earliest previous period (3 months') through the end of today. Periods are then
// computed from memory, so tab is instant (R-40).
func loadStats(d Deps) tea.Cmd {
	now := d.Clock.Now()
	return func() tea.Msg {
		var from, to time.Time
		for _, p := range []stats.Period{stats.Days7, stats.Days30, stats.Months3} {
			cur, prev, _ := stats.Window(p, now)
			if from.IsZero() || prev.Start.Before(from) {
				from = prev.Start
			}
			to = cur.End
		}
		ts, err := d.Store.Completed(d.Ctx, from, to)
		return statsLoadedMsg{tasks: ts, err: err}
	}
}

// exportedMsg reports the result of an export: the written path, or the error.
type exportedMsg struct {
	path string
	err  error
}

// runExport writes every task to a new export file (R-47, R-50).
func runExport(d Deps) tea.Cmd {
	now := d.Clock.Now()
	return func() tea.Msg {
		dir := d.ExportDir
		if dir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return exportedMsg{err: fmt.Errorf("finding the working directory: %w", err)}
			}
			dir = wd
		}
		ts, err := d.Store.All(d.Ctx)
		if err != nil {
			return exportedMsg{err: err}
		}
		path, err := export.Write(dir, ts, d.Version, now)
		return exportedMsg{path: path, err: err}
	}
}

// statsModel is the Stats screen (R-40 to R-47). It opens on 7 days every time (R-40).
type statsModel struct {
	period stats.Period
	loaded bool
	tasks  []task.Task
	err    string
	// exporting is true while an export runs; note is its result (R-47).
	exporting bool
	note      string // "Exported to <path>" or "Export failed: <error>"
	noteErr   bool
}

const exportedPrefix = "Exported to "

func (m statsModel) update(msg tea.Msg, d Deps) (statsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case statsLoadedMsg:
		m.loaded = true
		if msg.err != nil {
			d.Log.Error("loading stats", "err", msg.err)
			m.err = "Could not load stats: " + msg.err.Error()
			return m, nil
		}
		m.tasks = msg.tasks
	case exportedMsg:
		m.exporting = false
		if msg.err != nil {
			d.Log.Error("export", "err", msg.err)
			m.note, m.noteErr = "Export failed: "+msg.err.Error(), true
			return m, nil
		}
		d.Log.Info("exported", "path", msg.path)
		m.note, m.noteErr = exportedPrefix+msg.path, false
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			m.period = m.period.Next()
		case "e":
			if m.exporting {
				return m, nil
			}
			m.exporting, m.note = true, ""
			return m, runExport(d)
		}
	}
	return m, nil
}

var cardStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(28)

// view draws the period selector, the two scorecards (R-41, R-42), and the chart
// (R-43, R-45) sized to the terminal.
func (m statsModel) view(now time.Time, width, height int, th theme) string {
	s := stats.Compute(m.period, now, m.tasks)
	lines := []string{titleStyle.Render("Stats") + "  " + periodTabs(m.period), m.noteLine(width)}
	if m.err != "" {
		return lipgloss.JoinVertical(lipgloss.Left, append(lines, errText(0, m.err))...)
	}
	lines = append(lines,
		lipgloss.JoinHorizontal(lipgloss.Top,
			scorecard("Completed", countCard(s.Completed), th),
			" ",
			scorecard("Average per day", perDayCard(s.PerDay), th),
		),
		"",
		renderChart(s, width, height, th),
	)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// noteLine is the export result fitted to width, or blank. A long path keeps its
// label and loses its start (the file name at the end is the part worth reading); a
// long error loses its end.
func (m statsModel) noteLine(width int) string {
	if m.note == "" {
		return ""
	}
	if width <= 0 {
		width = chartDefWidth
	}
	text := []rune(m.note)
	if len(text) > width {
		if m.noteErr {
			text = append(text[:width-1:width-1], '…')
		} else {
			label := []rune(exportedPrefix)
			keep := width - len(label) - 1
			text = append(append(label, '…'), text[len(text)-keep:]...)
		}
	}
	if m.noteErr {
		return errStyle.Render(string(text))
	}
	return messageStyle.Render(string(text))
}

// periodTabs lists the periods with the current one highlighted.
func periodTabs(cur stats.Period) string {
	var b strings.Builder
	for i, p := range []stats.Period{stats.Days7, stats.Days30, stats.Months3} {
		if i > 0 {
			b.WriteString(" ")
		}
		style := dimStyle
		if p == cur {
			style = pickStyle
		}
		b.WriteString(style.Render(" " + p.String() + " "))
	}
	return b.String()
}

// card is a scorecard's text: the value, the signed change and percentage, the
// previous value, and the change's direction for colour.
type card struct {
	value, change, previous string
	sign                    int
}

// countCard formats the completed count (R-41): `+n`/`-n` and a whole percentage, `—`
// when the previous period had none.
func countCard(c stats.Comparison) card {
	cur, prev := int(c.Current), int(c.Previous)
	d := cur - prev
	return card{
		value:    fmt.Sprint(cur),
		change:   fmt.Sprintf("%+d (%s)", d, percent(c)),
		previous: fmt.Sprint(prev),
		sign:     sign(float64(d)),
	}
}

// perDayCard formats the average per day to one decimal (R-42). The change is between
// the shown (rounded) values, so the numbers on the card add up.
func perDayCard(c stats.Comparison) card {
	cur, prev := round1(c.Current), round1(c.Previous)
	d := round1(cur - prev)
	if d == 0 {
		d = 0 // no "-0.0"
	}
	return card{
		value:    fmt.Sprintf("%.1f", cur),
		change:   fmt.Sprintf("%+.1f (%s)", d, percent(c)),
		previous: fmt.Sprintf("%.1f", prev),
		sign:     sign(d),
	}
}

func percent(c stats.Comparison) string {
	p, ok := c.Percent()
	if !ok {
		return "—"
	}
	p = math.Round(p)
	if p == 0 {
		p = 0
	}
	return fmt.Sprintf("%+.0f%%", p)
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }

func sign(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// scorecard draws one card: title, value in bold, change (green up, red down), and the
// previous period's value.
func scorecard(title string, c card, th theme) string {
	change := lipgloss.NewStyle()
	if !th.noColor {
		switch c.sign {
		case 1:
			change = change.Foreground(lipgloss.Green)
		case -1:
			change = change.Foreground(lipgloss.Red)
		}
	}
	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		dimStyle.Render(title),
		titleStyle.Render(c.value),
		change.Render(c.change)+dimStyle.Render(" vs "+c.previous),
	))
}

func (m statsModel) hints() string { return "tab period · e export · esc back · ctrl+c quit" }
