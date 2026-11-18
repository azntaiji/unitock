// Package stats computes the Stats screen's periods, buckets, and scorecards (D-11,
// R-40 to R-44) as pure functions of the current time and the stored tasks.
//
// All dates are local to the location of the time passed in. Days are stepped by
// calendar date, never by 24 hours, so a DST change makes a day 23 or 25 hours long
// without moving any boundary off local midnight.
package stats

import (
	"time"

	"github.com/azntaiji/unitock/internal/task"
)

// Period is a Stats screen period (R-40).
type Period int

// The periods, in the order tab cycles them. Days7 is the default.
const (
	Days7 Period = iota
	Days30
	Months3
)

// Next returns the period after p, wrapping around (R-40).
func (p Period) Next() Period { return (p + 1) % 3 }

// String returns the period's display name.
func (p Period) String() string {
	switch p {
	case Days30:
		return "30 days"
	case Months3:
		return "3 months"
	default:
		return "7 days"
	}
}

// Label returns the x-axis label of a bucket starting at start (R-43): `ddd MMM DD` for
// 7 days, `MMM DD` for 30 days, `MMM YYYY` for 3 months (moment.js tokens).
func (p Period) Label(start time.Time) string {
	switch p {
	case Days30:
		return start.Format("Jan 02")
	case Months3:
		return start.Format("Jan 2006")
	default:
		return start.Format("Mon Jan 02")
	}
}

// Range is the half-open interval [Start, End) between two local midnights.
type Range struct {
	Start, End time.Time
}

// Contains reports whether t falls in r.
func (r Range) Contains(t time.Time) bool { return !t.Before(r.Start) && t.Before(r.End) }

// Days returns the number of calendar days in r.
func (r Range) Days() int {
	n := 0
	for d := r.Start; d.Before(r.End); d = addDays(d, 1) {
		n++
	}
	return n
}

// Window returns the current period, the previous period, and the current period's
// buckets for p at now (D-11).
func Window(p Period, now time.Time) (cur, prev Range, buckets []Range) {
	today := midnight(now)
	tomorrow := addDays(today, 1)
	switch p {
	case Days30:
		cur = Range{addDays(today, -29), tomorrow}
		prev = Range{addDays(cur.Start, -30), cur.Start}
		for s := cur.Start; s.Before(cur.End); s = addDays(s, 7) {
			buckets = append(buckets, Range{s, minTime(addDays(s, 7), cur.End)})
		}
	case Months3:
		first := time.Date(today.Year(), today.Month()-2, 1, 0, 0, 0, 0, today.Location())
		cur = Range{first, tomorrow}
		prev = Range{addMonths(first, -3), first}
		for s := first; s.Before(cur.End); s = addMonths(s, 1) {
			buckets = append(buckets, Range{s, minTime(addMonths(s, 1), cur.End)})
		}
	default:
		cur = Range{addDays(today, -6), tomorrow}
		prev = Range{addDays(cur.Start, -7), cur.Start}
		for s := cur.Start; s.Before(cur.End); s = addDays(s, 1) {
			buckets = append(buckets, Range{s, addDays(s, 1)})
		}
	}
	return cur, prev, buckets
}

// Bucket is one bar pair of the chart: the summed estimate and actual time of the
// tasks completed in its range (R-43).
type Bucket struct {
	Range
	Label     string
	Completed int
	Estimated time.Duration
	Actual    time.Duration
}

// Comparison is a scorecard value with the previous period's value (R-41, R-42).
type Comparison struct {
	Current, Previous float64
}

// Delta returns Current − Previous.
func (c Comparison) Delta() float64 { return c.Current - c.Previous }

// Percent returns the change as a percentage of Previous, and false when Previous is 0
// (shown as `—`).
func (c Comparison) Percent() (float64, bool) {
	if c.Previous == 0 {
		return 0, false
	}
	return c.Delta() / c.Previous * 100, true
}

// Summary is everything the Stats screen shows for one period.
type Summary struct {
	Period    Period
	Current   Range
	Previous  Range
	Buckets   []Bucket
	Completed Comparison // completed task count (R-41)
	PerDay    Comparison // average completed tasks per day (R-42)
}

// Empty reports whether no task was completed in the current period (R-45).
func (s Summary) Empty() bool { return s.Completed.Current == 0 }

// Compute summarises tasks for p at now. Only completed tasks count, attributed to the
// local date they ended (R-44); tasks may be passed in any order and may include
// active and cancelled ones.
func Compute(p Period, now time.Time, tasks []task.Task) Summary {
	cur, prev, ranges := Window(p, now)
	s := Summary{Period: p, Current: cur, Previous: prev, Buckets: make([]Bucket, len(ranges))}
	for i, r := range ranges {
		s.Buckets[i] = Bucket{Range: r, Label: p.Label(r.Start)}
	}
	var n, prevN int
	for _, t := range tasks {
		if t.Status != task.Completed {
			continue
		}
		end := t.EndedAt.In(now.Location())
		switch {
		case prev.Contains(end):
			prevN++
		case cur.Contains(end):
			n++
			for i := range s.Buckets {
				b := &s.Buckets[i]
				if b.Contains(end) {
					b.Completed++
					b.Estimated += t.Estimate()
					b.Actual += t.Elapsed(t.EndedAt)
					break
				}
			}
		}
	}
	// Average per day divides by the period's days; for 3 months only the days elapsed
	// so far (through today), which is cur.Days() since cur ends tonight (D-11).
	s.Completed = Comparison{float64(n), float64(prevN)}
	s.PerDay = Comparison{float64(n) / float64(cur.Days()), float64(prevN) / float64(prev.Days())}
	return s
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func addDays(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d+n, 0, 0, 0, 0, t.Location())
}

func addMonths(t time.Time, n int) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, t.Location())
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
