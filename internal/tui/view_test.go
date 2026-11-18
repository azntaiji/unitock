package tui

import (
	"image/color"
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
	dark := theme{dark: true}
	base := task.Task{ID: 1, Name: "Write report", Status: task.Active, StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	paused := base
	paused.Pauses = []task.Pause{{StartedAt: t0.Add(5 * time.Minute)}}
	long := base
	long.OriginalEstimate = 2 * time.Hour

	tests := []struct {
		name  string
		task  task.Task
		after time.Duration
		width int
		theme theme
	}{
		{"neutral", base, 5 * time.Minute, 80, dark},              // 80% left
		{"info", base, 12*time.Minute + 30*time.Second, 80, dark}, // exactly 50% left
		{"warning", base, 19 * time.Minute, 80, dark},             // 24% left
		{"error", base, 23 * time.Minute, 80, dark},               // 8% left
		{"overtime", base, 26*time.Minute + 5*time.Second, 80, dark},
		{"paused", paused, 10 * time.Minute, 80, dark}, // frozen at 20:00 left
		{"light", base, 19 * time.Minute, 80, theme{}},
		{"no_color_warning", base, 19 * time.Minute, 80, theme{noColor: true}},
		{"no_color_paused", paused, 10 * time.Minute, 80, theme{noColor: true}},
		{"hours", long, 10 * time.Minute, 80, dark},
		{"narrow", base, 5 * time.Minute, 50, dark},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newActiveModel(tt.task, 1)
			requireGolden(t, "active_"+tt.name, m.view(t0.Add(tt.after), tt.width, tt.theme))
		})
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

// The root model learns NO_COLOR from the colour profile and the background from the
// terminal, and passes both to the Active view.
func TestActiveViewTheme(t *testing.T) {
	h := newHarness(t)
	h.startTask("write", "25m")
	h.send(tea.ColorProfileMsg{Profile: colorprofile.ASCII})
	if !strings.Contains(h.view(), "[● RUNNING · ON TRACK]") {
		t.Errorf("NO_COLOR view lacks text badge:\n%s", h.view())
	}
	h.send(tea.ColorProfileMsg{Profile: colorprofile.TrueColor})
	h.send(tea.BackgroundColorMsg{Color: color.White})
	if h.m.theme != (theme{dark: false}) {
		t.Errorf("theme = %+v, want light with colour", h.m.theme)
	}
}
