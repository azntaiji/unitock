package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/unitock/internal/stats"
)

func TestScorecardCountFormat(t *testing.T) {
	tests := []struct {
		c                       stats.Comparison
		value, change, previous string
		sign                    int
	}{
		{stats.Comparison{Current: 6, Previous: 4}, "6", "+2 (+50%)", "4", 1},
		{stats.Comparison{Current: 1, Previous: 4}, "1", "-3 (-75%)", "4", -1},
		{stats.Comparison{Current: 3, Previous: 3}, "3", "+0 (+0%)", "3", 0},
		{stats.Comparison{Current: 5, Previous: 0}, "5", "+5 (—)", "0", 1},
		{stats.Comparison{Current: 0, Previous: 0}, "0", "+0 (—)", "0", 0},
		{stats.Comparison{Current: 2, Previous: 3}, "2", "-1 (-33%)", "3", -1},
	}
	for _, tc := range tests {
		got := countCard(tc.c)
		want := card{tc.value, tc.change, tc.previous, tc.sign}
		if got != want {
			t.Errorf("countCard(%v) = %+v, want %+v", tc.c, got, want)
		}
	}
}

func TestScorecardPerDayFormat(t *testing.T) {
	tests := []struct {
		c                       stats.Comparison
		value, change, previous string
	}{
		{stats.Comparison{Current: 12.0 / 7, Previous: 9.0 / 7}, "1.7", "+0.4 (+33%)", "1.3"},
		{stats.Comparison{Current: 1.0 / 7, Previous: 2.0 / 7}, "0.1", "-0.2 (-50%)", "0.3"},
		{stats.Comparison{Current: 3.0 / 30, Previous: 0}, "0.1", "+0.1 (—)", "0.0"},
		{stats.Comparison{Current: 0.04, Previous: 0.06}, "0.0", "-0.1 (-33%)", "0.1"},
		{stats.Comparison{Current: 0.5, Previous: 0.5}, "0.5", "+0.0 (+0%)", "0.5"},
	}
	for _, tc := range tests {
		got := perDayCard(tc.c)
		if got.value != tc.value || got.change != tc.change || got.previous != tc.previous {
			t.Errorf("perDayCard(%v) = %+v, want %s %s %s", tc.c, got, tc.value, tc.change, tc.previous)
		}
	}
}

// The scorecards count only completed tasks, loaded from the store (R-41, R-42, R-44).
func TestScorecardOnlyCompletedCount(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	end := func(cancel bool, startedAgo time.Duration) {
		t.Helper()
		tk, err := h.store.Create(ctx, "", "x", time.Hour, t0.Add(-startedAgo))
		if err != nil {
			t.Fatal(err)
		}
		at := t0.Add(-startedAgo + 30*time.Minute)
		if cancel {
			err = h.store.Cancel(ctx, tk.ID, at)
		} else {
			err = h.store.Complete(ctx, tk.ID, at)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	day := 24 * time.Hour
	end(false, time.Hour)                                                    // this period
	end(false, 2*day)                                                        // this period
	end(true, 3*day)                                                         // cancelled: not counted
	end(false, 8*day)                                                        // previous period
	if _, err := h.store.Create(ctx, "", "now", time.Hour, t0); err != nil { // active
		t.Fatal(err)
	}

	var m statsModel
	m, _ = m.update(loadStats(h.m.deps)(), h.m.deps)
	if !m.loaded || m.err != "" {
		t.Fatalf("loaded=%v err=%q", m.loaded, m.err)
	}
	s := stats.Compute(m.period, t0, m.tasks)
	if s.Completed != (stats.Comparison{Current: 2, Previous: 1}) {
		t.Errorf("completed = %+v, want {2 1}", s.Completed)
	}
	v := ansi.Strip(m.view(t0, 60, 20, theme{noColor: true}))
	for _, want := range []string{"Completed", "+1 (+100%) vs 1", "Average per day", "0.3", "+0.2 (+100%) vs 0.1"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
}

func TestScorecardLoadError(t *testing.T) {
	var m statsModel
	h := newHarness(t)
	m, _ = m.update(statsLoadedMsg{err: context.Canceled}, h.m.deps)
	if v := m.view(t0, 60, 20, theme{}); !strings.Contains(v, "Could not load stats") {
		t.Errorf("view lacks the load error:\n%s", v)
	}
}
