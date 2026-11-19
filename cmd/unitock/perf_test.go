package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/store/storetest"
	"github.com/azntaiji/unitock/internal/tui"
)

// Startup budget with 10,000 tasks (SPEC Constraints, P4-02).
const (
	perfTasks     = 10_000
	startupBudget = 300 * time.Millisecond
	perfRuns      = 5
)

// TestPerfStartup times run from its call to the first rendered frame, with 10,000 tasks
// in the database: flag parsing, data dir, log, lock, opening and migrating the store,
// building the UI, and its first View. The UI runs without a terminal, so process start
// and the terminal write are not included; both are a few milliseconds. The slowest of
// several runs must be within budget.
func TestPerfStartup(t *testing.T) {
	if storetest.Race {
		t.Skip("timings are meaningless under the race detector")
	}
	for _, tc := range []struct {
		name   string
		active bool   // relaunch into a running task in overtime (R-05)
		want   string // on the first frame
	}{{"start screen", false, "Start task"}, {"active task", true, "running"}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			now := time.Now()
			if err := storetest.Seed(ctx, dir, perfTasks, 180*24*time.Hour, now.Add(-3*time.Hour)); err != nil {
				t.Fatal(err)
			}
			if tc.active {
				st, err := store.Open(ctx, dir)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := st.Create(ctx, "Project 01", "running", time.Hour, now.Add(-2*time.Hour)); err != nil {
					t.Fatal(err)
				}
				_ = st.Close()
			}

			var began time.Time
			var frame string
			var took time.Duration
			stubUI(t, func(ctx context.Context, d deps) error {
				m, err := tui.New(tui.Deps{Ctx: ctx, Store: d.store, Clock: d.clock, Log: d.log, Version: "perf"})
				if err != nil {
					return err
				}
				next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				frame = next.View().Content
				took = time.Since(began)
				return nil
			})

			var worst time.Duration
			for i := range perfRuns {
				var out, errb bytes.Buffer
				began = time.Now()
				if code := run(ctx, []string{"--data-dir", dir}, &out, &errb, noEnv); code != 0 {
					t.Fatalf("exit = %d, stderr %q", code, errb.String())
				}
				if !strings.Contains(frame, tc.want) {
					t.Fatalf("first frame lacks %q:\n%s", tc.want, frame)
				}
				t.Logf("run %d: startup to first frame %v", i+1, took)
				worst = max(worst, took)
			}
			t.Logf("slowest: %v (budget %v)", worst, startupBudget)
			if worst >= startupBudget {
				t.Errorf("startup to first frame %v, want under %v", worst, startupBudget)
			}
		})
	}
}
