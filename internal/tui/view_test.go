package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/azntaiji/unitock/internal/task"
)

// requireGolden compares got with testdata/<name>.golden. `make golden` (UPDATE_GOLDEN=1)
// rewrites the file instead. On a mismatch the error shows both sides with escape
// sequences quoted, so colour changes are visible.
func requireGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run `make golden` to create it)", err)
	}
	if string(want) != got {
		t.Errorf("view differs from %s\n--- got\n%s\n--- want\n%s", path, quoteLines(got), quoteLines(string(want)))
	}
}

func quoteLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strconv.Quote(l)
	}
	return strings.Join(lines, "\n")
}

// TestActiveView renders the Active screen in each D-09 band and state and compares it
// with reviewed golden files (P2-01). The task is 25 minutes from t0.
func TestActiveView(t *testing.T) {
	var col theme
	base := task.Task{ID: 1, Name: "Write report", Project: "unitock", Status: task.Active, StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	paused := base
	paused.Pauses = []task.Pause{{StartedAt: t0.Add(5 * time.Minute)}}
	long := base
	long.OriginalEstimate = 2 * time.Hour
	noProject := base
	noProject.Project = ""
	extended := base
	extended.Extensions = []task.Extension{{At: t0.Add(20 * time.Minute), Added: 10 * time.Minute}}

	tests := []struct {
		name  string
		task  task.Task
		after time.Duration
		width int
		theme theme
	}{
		{"neutral", base, 5 * time.Minute, 80, col},              // 80% left, RUNNING
		{"info", base, 12*time.Minute + 30*time.Second, 80, col}, // exactly 50% left, RUNNING
		{"warning", base, 19 * time.Minute, 80, col},             // 24% left, ENDING
		{"error", base, 23 * time.Minute, 80, col},               // 8% left, ENDING
		{"overtime", base, 26*time.Minute + 5*time.Second, 80, col},
		{"paused", paused, 10 * time.Minute, 80, col}, // frozen at 20:00 left
		{"no_project", noProject, 5 * time.Minute, 80, col},
		{"no_color_warning", base, 19 * time.Minute, 80, theme{noColor: true}},
		{"no_color_paused", paused, 10 * time.Minute, 80, theme{noColor: true}},
		{"hours", long, 10 * time.Minute, 80, col},
		{"narrow", base, 5 * time.Minute, 50, col},
		{"extended", extended, 22 * time.Minute, 80, col}, // 25:00 + 10:00, 13:00 left
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newActiveModel(tt.task, 1)
			requireGolden(t, "active_"+tt.name, m.view(t0.Add(tt.after), tt.width, tt.theme))
		})
	}
}

// The row under the bar shows elapsed and the original estimate, adds the total added
// time once extended, and never repeats the remaining time (R-17, P4-05).
func TestActiveViewTimeRow(t *testing.T) {
	base := task.Task{StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	ext := base
	ext.Extensions = []task.Extension{{At: t0.Add(5 * time.Minute), Added: 10 * time.Minute}, {At: t0.Add(30 * time.Minute), Added: 90 * time.Second}}
	paused := base
	paused.Pauses = []task.Pause{{StartedAt: t0.Add(5 * time.Minute), EndedAt: t0.Add(15 * time.Minute)}}
	long := base
	long.OriginalEstimate = 90 * time.Minute
	for _, tt := range []struct {
		name  string
		t     task.Task
		after time.Duration
		want  string
	}{
		{"start", base, 0, "Elapsed 00:00 · Estimate 25:00"},
		{"rounds elapsed down", base, 12*time.Minute + 30*time.Second + 900*time.Millisecond, "Elapsed 12:30 · Estimate 25:00"},
		{"paused time not counted", paused, 20 * time.Minute, "Elapsed 10:00 · Estimate 25:00"},
		{"overtime keeps counting", base, 27 * time.Minute, "Elapsed 27:00 · Estimate 25:00"},
		{"extensions summed", ext, 31 * time.Minute, "Elapsed 31:00 · Estimate 25:00 + 11:30 added"},
		{"hours", long, 65 * time.Minute, "Elapsed 1:05:00 · Estimate 1:30:00"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeRow(tt.t, t0.Add(tt.after)); got != tt.want {
				t.Errorf("timeRow = %q, want %q", got, tt.want)
			}
		})
	}
	view := newActiveModel(base, 1).view(t0.Add(5*time.Minute), 80, theme{})
	if strings.Contains(view, "Remaining") || strings.Contains(view, "Overtime ") {
		t.Errorf("view repeats the remaining time:\n%s", view)
	}
}

// The badge follows D-09: PAUSED overrides, OVERTIME past the estimate, ENDING at 25%
// or less left, otherwise RUNNING.
func TestActiveViewBadge(t *testing.T) {
	tk := task.Task{StartedAt: t0, OriginalEstimate: 100 * time.Second}
	paused := tk
	paused.Pauses = []task.Pause{{StartedAt: t0.Add(99 * time.Second)}}
	for _, tt := range []struct {
		t     task.Task
		after time.Duration
		want  string
	}{
		{tk, 0, "● RUNNING"},
		{tk, 74 * time.Second, "● RUNNING"},
		{tk, 75 * time.Second, "◐ ENDING"},
		{tk, 100 * time.Second, "◐ ENDING"},
		{tk, 101 * time.Second, "▲ OVERTIME"},
		{paused, 200 * time.Second, "❚❚ PAUSED"},
	} {
		now := t0.Add(tt.after)
		if got := badge(tt.t, bandAt(tt.t, now), theme{}); !strings.Contains(got, tt.want) {
			t.Errorf("+%v: badge %q, want %q", tt.after, got, tt.want)
		}
	}
}

func TestActiveViewBands(t *testing.T) {
	tk := task.Task{StartedAt: t0, OriginalEstimate: 100 * time.Second}
	tests := []struct {
		after time.Duration
		want  band
	}{
		{0, bandNeutral},
		{49 * time.Second, bandNeutral},
		{50 * time.Second, bandInfo}, // 50% left is info
		{74 * time.Second, bandInfo},
		{75 * time.Second, bandWarning}, // 25% left
		{89 * time.Second, bandWarning},
		{90 * time.Second, bandError}, // 10% left
		{100 * time.Second, bandError},
		{100*time.Second + 1, bandOvertime},
	}
	for _, tt := range tests {
		if got := bandAt(tk, t0.Add(tt.after)); got != tt.want {
			t.Errorf("bandAt(+%v) = %v, want %v", tt.after, got, tt.want)
		}
	}
}

func TestActiveViewProgressBar(t *testing.T) {
	tk := task.Task{StartedAt: t0, OriginalEstimate: 100 * time.Second}
	for _, tt := range []struct {
		after  time.Duration
		filled int
		pct    string
	}{
		{0, 0, "  0%"},
		{50 * time.Second, 20, " 50%"},
		{100 * time.Second, 40, "100%"},
		{150 * time.Second, 40, "150%"}, // overtime: full bar, percentage keeps counting
	} {
		got := progressBar(tk, t0.Add(tt.after), bandNeutral.color(theme{noColor: true}))
		if n := strings.Count(got, "█"); n != tt.filled || !strings.HasSuffix(got, tt.pct) {
			t.Errorf("+%v: %d filled, %q; want %d filled, suffix %q", tt.after, n, got, tt.filled, tt.pct)
		}
	}
}

// The root model learns NO_COLOR from the colour profile and passes it to the view.
func TestActiveViewTheme(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.send(tea.ColorProfileMsg{Profile: colorprofile.ASCII})
	if !h.m.theme.noColor || strings.Contains(h.view(), "\x1b[1;30;42m") {
		t.Errorf("NO_COLOR not applied: %q", h.view())
	}
	h.send(tea.ColorProfileMsg{Profile: colorprofile.ANSI256})
	if h.m.theme.noColor {
		t.Error("colour profile did not restore colour")
	}
}
