// Package task holds the task model and its timer maths (D-07). Nothing is ticked or
// accumulated in memory: every value is derived from stored timestamps and the time
// passed in by the caller, so time while the app is closed or the machine sleeps counts
// as elapsed unless the task is paused (R-04).
package task

import (
	"slices"
	"time"
)

// Status is a task's lifecycle state (R-03).
type Status string

// Task statuses.
const (
	Active    Status = "active"
	Completed Status = "completed"
	Cancelled Status = "cancelled"
)

// Valid reports whether s is a known status.
func (s Status) Valid() bool {
	switch s {
	case Active, Completed, Cancelled:
		return true
	}
	return false
}

// Extension is time added to a task's estimate.
type Extension struct {
	At    time.Time
	Added time.Duration
}

// Pause is an interval during which a task's timer is stopped. A zero EndedAt means the
// pause is still open.
type Pause struct {
	StartedAt time.Time
	EndedAt   time.Time
}

// Open reports whether the pause has not ended.
func (p Pause) Open() bool { return p.EndedAt.IsZero() }

// Task is one timed task with its extensions and pauses. A zero EndedAt means the task
// has not ended.
type Task struct {
	ID               int64
	Name             string
	Project          string // empty when the task has no project (R-25)
	Status           Status
	StartedAt        time.Time
	EndedAt          time.Time
	OriginalEstimate time.Duration
	Extensions       []Extension
	Pauses           []Pause
}

// Estimate is the original estimate plus every extension.
func (t Task) Estimate() time.Duration {
	d := t.OriginalEstimate
	for _, e := range t.Extensions {
		d += e.Added
	}
	return d
}

// Paused reports whether the task has an open pause.
func (t Task) Paused() bool {
	for _, p := range t.Pauses {
		if p.Open() {
			return true
		}
	}
	return false
}

// Elapsed is (EndedAt or now) − StartedAt − paused time, where an open pause runs to the
// same end point. Pauses are clipped to the task's span, and the result is never
// negative (for example if the clock moves back before StartedAt).
func (t Task) Elapsed(now time.Time) time.Duration {
	end := t.end(now)
	if !end.After(t.StartedAt) {
		return 0
	}
	d := end.Sub(t.StartedAt) - t.pausedWithin(t.StartedAt, end)
	return max(d, 0)
}

// Remaining is the estimate minus elapsed time, floored at zero.
func (t Task) Remaining(now time.Time) time.Duration {
	return max(t.Estimate()-t.Elapsed(now), 0)
}

// Overtime is elapsed time beyond the estimate, floored at zero.
func (t Task) Overtime(now time.Time) time.Duration {
	return max(t.Elapsed(now)-t.Estimate(), 0)
}

// ExtensionFor is the amount to add to the estimate when the user enters entered at now
// (D-10). Before time is up, or from the `d` key, it is exactly entered. From the
// time-up alert it also covers the overtime so far, so remaining becomes entered,
// counted from now.
func (t Task) ExtensionFor(now time.Time, entered time.Duration, fromAlert bool) time.Duration {
	if fromAlert {
		return t.Overtime(now) + entered
	}
	return entered
}

// TimeUp reports whether elapsed time has reached the estimate.
func (t Task) TimeUp(now time.Time) bool {
	return t.Elapsed(now) >= t.Estimate()
}

func (t Task) end(now time.Time) time.Time {
	if !t.EndedAt.IsZero() {
		return t.EndedAt
	}
	return now
}

// pausedWithin sums pause time inside [from, to]. Overlapping pauses are counted once.
func (t Task) pausedWithin(from, to time.Time) time.Duration {
	var total time.Duration
	covered := from // pause time before this instant is already counted
	for _, p := range sortedPauses(t.Pauses) {
		start := p.StartedAt
		stop := p.EndedAt
		if p.Open() || stop.After(to) {
			stop = to
		}
		if start.Before(covered) {
			start = covered
		}
		if stop.After(start) {
			total += stop.Sub(start)
			covered = stop
		}
	}
	return total
}

// sortedPauses returns the pauses ordered by start time without modifying the input.
func sortedPauses(ps []Pause) []Pause {
	out := slices.Clone(ps)
	slices.SortFunc(out, func(a, b Pause) int { return a.StartedAt.Compare(b.StartedAt) })
	return out
}
