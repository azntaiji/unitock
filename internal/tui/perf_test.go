package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/unitock/internal/stats"
	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/store/storetest"
	"github.com/azntaiji/unitock/internal/task"
)

// Stats render budget with 10,000 tasks (SPEC Constraints, P4-02).
const (
	perfTasks   = 10_000
	statsBudget = 100 * time.Millisecond
	perfRuns    = 5
)

// TestPerfStats times the Stats screen with 10,000 tasks, all inside the periods it
// reads, so every one is loaded: from `s` to the rendered frame (reading the store,
// computing the period, drawing the chart), then each tab to the next period. The
// slowest of several runs must be within budget.
func TestPerfStats(t *testing.T) {
	if storetest.Race {
		t.Skip("timings are meaningless under the race detector")
	}
	ctx := context.Background()
	dir := t.TempDir()
	_, prev, _ := stats.Window(stats.Months3, t0) // the earliest period Stats reads
	span := t0.Sub(prev.Start) - 24*time.Hour     // and a day's margin
	if err := storetest.Seed(ctx, dir, perfTasks, span, t0); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	m, err := New(Deps{Store: st, Clock: task.NewFakeClock(t0)})
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(Model)

	var worstOpen, worstTab time.Duration
	for i := range perfRuns {
		began := time.Now()
		opened, cmd := m.openStats()
		next, _ := opened.Update(cmd())
		s := next.(Model)
		frame := s.View().Content
		took := time.Since(began)
		if !s.stats.loaded || s.stats.err != "" {
			t.Fatalf("stats not loaded: %q", s.stats.err)
		}
		if want := perfTasks - perfTasks/20; len(s.stats.tasks) != want { // all completed tasks
			t.Fatalf("loaded %d tasks, want %d", len(s.stats.tasks), want)
		}
		if !strings.Contains(frame, "7 days") {
			t.Fatalf("frame lacks the 7 days tab:\n%s", frame)
		}
		t.Logf("run %d: open Stats %v", i+1, took)
		worstOpen = max(worstOpen, took)

		for range 2 { // 30 days, then 3 months
			began = time.Now()
			next, _ = s.Update(key("tab"))
			s = next.(Model)
			_ = s.View()
			took = time.Since(began)
			t.Logf("run %d: tab to %v %v", i+1, s.stats.period, took)
			worstTab = max(worstTab, took)
		}
	}
	t.Logf("slowest: open %v, tab %v (budget %v)", worstOpen, worstTab, statsBudget)
	if worstOpen >= statsBudget || worstTab >= statsBudget {
		t.Errorf("stats render: open %v, tab %v; want each under %v", worstOpen, worstTab, statsBudget)
	}
}
