package task

import (
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func at(d time.Duration) time.Time { return t0.Add(d) }

func TestTimerMaths(t *testing.T) {
	const m = time.Minute
	tests := []struct {
		name                         string
		task                         Task
		now                          time.Time
		elapsed, remaining, overtime time.Duration
		paused, timeUp               bool
	}{
		{
			name:    "just started",
			task:    Task{StartedAt: t0, OriginalEstimate: 30 * m},
			now:     t0,
			elapsed: 0, remaining: 30 * m,
		},
		{
			name:    "running",
			task:    Task{StartedAt: t0, OriginalEstimate: 30 * m},
			now:     at(10*m + 5*time.Second),
			elapsed: 10*m + 5*time.Second, remaining: 19*m + 55*time.Second,
		},
		{
			// R-04: the task was started in the past, and the app was closed (or the
			// machine slept) for most of it. Wall-clock time still counts.
			name:    "started in the past, app closed meanwhile",
			task:    Task{StartedAt: t0, OriginalEstimate: 2 * time.Hour},
			now:     at(95 * m),
			elapsed: 95 * m, remaining: 25 * m,
		},
		{
			name:    "ran out while the app was closed",
			task:    Task{StartedAt: t0, OriginalEstimate: 30 * m},
			now:     at(42 * m),
			elapsed: 42 * m, overtime: 12 * m, timeUp: true,
		},
		{
			name:    "exactly at the estimate",
			task:    Task{StartedAt: t0, OriginalEstimate: 30 * m},
			now:     at(30 * m),
			elapsed: 30 * m, timeUp: true,
		},
		{
			name: "extensions add to the estimate",
			task: Task{StartedAt: t0, OriginalEstimate: 30 * m, Extensions: []Extension{
				{At: at(20 * m), Added: 10 * m},
				{At: at(35 * m), Added: 5 * m},
			}},
			now:     at(40 * m),
			elapsed: 40 * m, remaining: 5 * m,
		},
		{
			name: "closed pause is not elapsed",
			task: Task{StartedAt: t0, OriginalEstimate: 30 * m, Pauses: []Pause{
				{StartedAt: at(5 * m), EndedAt: at(15 * m)},
			}},
			now:     at(25 * m),
			elapsed: 15 * m, remaining: 15 * m,
		},
		{
			// R-05: a task paused at quit is still paused at launch, and the time the
			// app was closed during the pause does not count.
			name: "open pause runs to now",
			task: Task{StartedAt: t0, OriginalEstimate: 30 * m, Pauses: []Pause{
				{StartedAt: at(10 * m), EndedAt: at(12 * m)},
				{StartedAt: at(20 * m)},
			}},
			now:     at(3 * time.Hour),
			elapsed: 18 * m, remaining: 12 * m, paused: true,
		},
		{
			name: "completed task uses ended_at, not now",
			task: Task{
				Status: Completed, StartedAt: t0, EndedAt: at(50 * m), OriginalEstimate: 45 * m,
				Pauses: []Pause{{StartedAt: at(10 * m), EndedAt: at(20 * m)}},
			},
			now:     at(10 * time.Hour),
			elapsed: 40 * m, remaining: 5 * m,
		},
		{
			name: "pause left open on an ended task stops at ended_at",
			task: Task{
				Status: Cancelled, StartedAt: t0, EndedAt: at(30 * m), OriginalEstimate: 60 * m,
				Pauses: []Pause{{StartedAt: at(20 * m)}},
			},
			now:     at(5 * time.Hour),
			elapsed: 20 * m, remaining: 40 * m, paused: true,
		},
		{
			name: "overlapping and unordered pauses count once",
			task: Task{StartedAt: t0, OriginalEstimate: 60 * m, Pauses: []Pause{
				{StartedAt: at(20 * m), EndedAt: at(30 * m)},
				{StartedAt: at(5 * m), EndedAt: at(10 * m)},
				{StartedAt: at(25 * m), EndedAt: at(35 * m)},
			}},
			now:     at(50 * m),
			elapsed: 30 * m, remaining: 30 * m,
		},
		{
			name: "pause outside the task span is clipped",
			task: Task{StartedAt: t0, OriginalEstimate: 60 * m, Pauses: []Pause{
				{StartedAt: at(-10 * m), EndedAt: at(5 * m)},
			}},
			now:     at(20 * m),
			elapsed: 15 * m, remaining: 45 * m,
		},
		{
			name:    "clock before start gives zero elapsed",
			task:    Task{StartedAt: t0, OriginalEstimate: 30 * m},
			now:     at(-time.Hour),
			elapsed: 0, remaining: 30 * m,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.Elapsed(tt.now); got != tt.elapsed {
				t.Errorf("Elapsed = %v, want %v", got, tt.elapsed)
			}
			if got := tt.task.Remaining(tt.now); got != tt.remaining {
				t.Errorf("Remaining = %v, want %v", got, tt.remaining)
			}
			if got := tt.task.Overtime(tt.now); got != tt.overtime {
				t.Errorf("Overtime = %v, want %v", got, tt.overtime)
			}
			if got := tt.task.Paused(); got != tt.paused {
				t.Errorf("Paused = %v, want %v", got, tt.paused)
			}
			if got := tt.task.TimeUp(tt.now); got != tt.timeUp {
				t.Errorf("TimeUp = %v, want %v", got, tt.timeUp)
			}
		})
	}
}

func TestElapsedAdvancesWithInjectedClock(t *testing.T) {
	clk := NewFakeClock(t0)
	tk := Task{StartedAt: clk.Now(), OriginalEstimate: time.Minute}
	for i := 1; i <= 75; i++ {
		clk.Advance(time.Second)
		want := time.Duration(i) * time.Second
		if got := tk.Elapsed(clk.Now()); got != want {
			t.Fatalf("after %d ticks Elapsed = %v, want %v", i, got, want)
		}
	}
	if got := tk.Overtime(clk.Now()); got != 15*time.Second {
		t.Errorf("Overtime = %v, want 15s", got)
	}
}

func TestPausedWithinDoesNotReorderInput(t *testing.T) {
	ps := []Pause{{StartedAt: at(20 * time.Minute)}, {StartedAt: at(time.Minute), EndedAt: at(2 * time.Minute)}}
	tk := Task{StartedAt: t0, Pauses: ps}
	_ = tk.Elapsed(at(time.Hour))
	if !ps[0].StartedAt.Equal(at(20 * time.Minute)) {
		t.Error("Elapsed reordered the caller's pauses")
	}
}

func TestStatusValid(t *testing.T) {
	for _, s := range []Status{Active, Completed, Cancelled} {
		if !s.Valid() {
			t.Errorf("%q should be valid", s)
		}
	}
	if Status("paused").Valid() {
		t.Error(`"paused" should not be a status`)
	}
}

func TestFakeClockConcurrent(t *testing.T) {
	clk := NewFakeClock(t0)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				clk.Advance(time.Millisecond)
				_ = clk.Now()
			}
		})
	}
	wg.Wait()
	if got := clk.Now(); !got.Equal(at(800 * time.Millisecond)) {
		t.Errorf("Now = %v, want %v", got, at(800*time.Millisecond))
	}
	clk.Set(t0)
	if !clk.Now().Equal(t0) {
		t.Error("Set did not move the clock")
	}
}

// D-10: extending from the alert covers the overtime so far; otherwise exactly entered.
func TestExtendExtensionFor(t *testing.T) {
	t0 := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	tk := Task{StartedAt: t0, OriginalEstimate: 25 * time.Minute}
	early, late := t0.Add(10*time.Minute), t0.Add(28*time.Minute) // 3m overtime at late
	for _, tt := range []struct {
		now       time.Time
		fromAlert bool
		want      time.Duration
	}{
		{early, false, 10 * time.Minute},
		{early, true, 10 * time.Minute}, // no overtime yet
		{late, false, 10 * time.Minute},
		{late, true, 13 * time.Minute},
	} {
		if got := tk.ExtensionFor(tt.now, 10*time.Minute, tt.fromAlert); got != tt.want {
			t.Errorf("ExtensionFor(%v, alert=%v) = %v, want %v", tt.now.Sub(t0), tt.fromAlert, got, tt.want)
		}
	}
	ext := tk
	ext.Extensions = []Extension{{At: late, Added: tk.ExtensionFor(late, 10*time.Minute, true)}}
	if r := ext.Remaining(late); r != 10*time.Minute {
		t.Errorf("remaining after alert extend = %v, want 10m from now", r)
	}
}
