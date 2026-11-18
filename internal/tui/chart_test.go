package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/unitock/internal/stats"
	"github.com/azntaiji/unitock/internal/task"
)

// chartTasks is a fixed history ending at t0 (Tue 2026-11-17 09:00 UTC): a few tasks a
// day over the last week, then one every third day back to mid-August, with actual
// sometimes over and sometimes under the estimate.
func chartTasks() []task.Task {
	var ts []task.Task
	add := func(end time.Time, est, act time.Duration) {
		ts = append(ts, task.Task{Status: task.Completed, StartedAt: end.Add(-act), EndedAt: end, OriginalEstimate: est})
	}
	week := [][2]time.Duration{
		{45 * time.Minute, 50 * time.Minute}, {time.Hour, 40 * time.Minute},
		{2 * time.Hour, 2*time.Hour + 30*time.Minute}, {30 * time.Minute, 20 * time.Minute},
		{90 * time.Minute, 100 * time.Minute}, {0, 0}, {25 * time.Minute, 25 * time.Minute},
	}
	for i, d := range week {
		if d[0] == 0 {
			continue // a day off: empty bucket
		}
		end := t0.AddDate(0, 0, i-6).Add(-time.Hour)
		add(end, d[0], d[1])
		add(end.Add(-3*time.Hour), d[0]/2, d[1]/3)
	}
	for d := 9; d < 95; d += 3 {
		add(t0.AddDate(0, 0, -d), time.Hour, time.Duration(30+d%60)*time.Minute)
	}
	return ts
}

// P3-03: the chart per period, wide, NO_COLOR, and empty, against reviewed goldens
// (R-43, R-45). The whole Stats screen is rendered, as the user sees it.
func TestChartGolden(t *testing.T) {
	color, plain := theme{}, theme{noColor: true}
	tests := []struct {
		name   string
		period stats.Period
		w, h   int
		th     theme
		tasks  []task.Task
	}{
		{"stats_7days", stats.Days7, 60, 20, color, chartTasks()},
		{"stats_30days", stats.Days30, 60, 20, color, chartTasks()},
		{"stats_3months", stats.Months3, 60, 20, color, chartTasks()},
		{"stats_7days_wide", stats.Days7, 100, 30, color, chartTasks()},
		{"stats_7days_no_color", stats.Days7, 60, 20, plain, chartTasks()},
		{"stats_empty", stats.Days7, 60, 20, color, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := statsModel{period: tc.period, loaded: true, tasks: tc.tasks}
			requireGolden(t, tc.name, m.view(t0, tc.w, tc.h, tc.th))
		})
	}
}

// The Stats screen fits the minimum terminal for every period, with or without data,
// leaving the blank line and hint bar the root adds (R-08).
func TestChartFitsMinimumTerminal(t *testing.T) {
	for _, tasks := range [][]task.Task{chartTasks(), nil} {
		for _, p := range []stats.Period{stats.Days7, stats.Days30, stats.Months3} {
			m := statsModel{period: p, loaded: true, tasks: tasks}
			v := m.view(t0, MinWidth, MinHeight, theme{})
			if w, h := lipgloss.Width(v), lipgloss.Height(v)+2; w > MinWidth || h > MinHeight {
				t.Errorf("%v (data %v): %d×%d, larger than %d×%d:\n%s", p, tasks != nil, w, h, MinWidth, MinHeight, v)
			}
		}
	}
}

// Labels follow R-43 for each period.
func TestChartLabels(t *testing.T) {
	for _, tc := range []struct {
		p    stats.Period
		want []string
	}{
		{stats.Days7, []string{"Wed", "Nov 11", "Tue", "Nov 17"}},
		{stats.Days30, []string{"Oct 19", "Nov 16"}},
		{stats.Months3, []string{"Sep 2026", "Oct 2026", "Nov 2026"}},
	} {
		m := statsModel{period: tc.p, loaded: true, tasks: chartTasks()}
		v := ansi.Strip(m.view(t0, 60, 20, theme{}))
		for _, w := range tc.want {
			if !strings.Contains(v, w) {
				t.Errorf("%v chart lacks label %q:\n%s", tc.p, w, v)
			}
		}
	}
	// Wide enough, 7-day labels stay on one line as `ddd MMM DD`.
	m := statsModel{period: stats.Days7, loaded: true, tasks: chartTasks()}
	if v := ansi.Strip(m.view(t0, 100, 30, theme{})); !strings.Contains(v, "Tue Nov 17") {
		t.Errorf("wide 7-day chart lacks \"Tue Nov 17\":\n%s", v)
	}
}

func TestChartEmptyState(t *testing.T) {
	m := statsModel{loaded: true}
	v := ansi.Strip(m.view(t0, 60, 20, theme{}))
	if !strings.Contains(v, "No completed tasks in this period.") || strings.Contains(v, "└") {
		t.Errorf("empty state wrong:\n%s", v)
	}
}

func TestChartNiceTopAndFormat(t *testing.T) {
	for _, tc := range []struct {
		in   time.Duration
		top  string
		half string
	}{
		{time.Minute, "10m", "5m"},
		{25 * time.Minute, "30m", "15m"},
		{50 * time.Minute, "1h", "30m"},
		{3*time.Hour + time.Second, "4h", "2h"},
		{5 * time.Hour, "6h", "3h"},
		{3 * time.Hour, "3h", "1h30m"},
		{20 * time.Hour, "24h", "12h"},
		{30 * time.Hour, "32h", "16h"},
		{40 * time.Hour, "48h", "24h"},
		{150 * time.Hour, "192h", "96h"},
	} {
		top := niceTop(tc.in)
		if got := formatHM(top); got != tc.top || formatHM(top/2) != tc.half {
			t.Errorf("niceTop(%v) = %s / %s, want %s / %s", tc.in, got, formatHM(top/2), tc.top, tc.half)
		}
	}
}

func TestChartBarCells(t *testing.T) {
	for _, tc := range []struct {
		d, top time.Duration
		rows   int
		want   int
	}{
		{0, time.Hour, 6, 0},
		{time.Second, time.Hour, 6, 1}, // any time shows
		{30 * time.Minute, time.Hour, 6, 3},
		{time.Hour, time.Hour, 6, 6},
		{24 * time.Minute, time.Hour, 6, 2}, // 2.4 rounds down
		{27 * time.Minute, time.Hour, 6, 3}, // 2.7 rounds up
	} {
		if got := barCells(tc.d, tc.top, tc.rows); got != tc.want {
			t.Errorf("barCells(%v, %v, %d) = %d, want %d", tc.d, tc.top, tc.rows, got, tc.want)
		}
	}
}
