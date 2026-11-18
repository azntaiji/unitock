package stats

import (
	"slices"
	"testing"
	"time"
	_ "time/tzdata" // fixed zone data so DST tests do not depend on the host

	"github.com/azntaiji/unitock/internal/task"
)

var la = mustLoad("America/Los_Angeles")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, la)
}

func day(y int, m time.Month, d int) time.Time { return at(y, m, d, 0, 0) }

func TestWindow(t *testing.T) {
	type want struct {
		cur, prev   Range
		bucketStart []time.Time
		lastEnd     time.Time
		labels      []string
		curDays     int
		prevDays    int
	}
	tests := []struct {
		name string
		p    Period
		now  time.Time
		want want
	}{
		{
			name: "7 days ordinary week",
			p:    Days7, now: at(2026, 11, 18, 15, 4),
			want: want{
				cur:  Range{day(2026, 11, 12), day(2026, 11, 19)},
				prev: Range{day(2026, 11, 5), day(2026, 11, 12)},
				bucketStart: []time.Time{day(2026, 11, 12), day(2026, 11, 13), day(2026, 11, 14),
					day(2026, 11, 15), day(2026, 11, 16), day(2026, 11, 17), day(2026, 11, 18)},
				lastEnd: day(2026, 11, 19),
				labels: []string{"Thu Nov 12", "Fri Nov 13", "Sat Nov 14", "Sun Nov 15",
					"Mon Nov 16", "Tue Nov 17", "Wed Nov 18"},
				curDays: 7, prevDays: 7,
			},
		},
		{
			name: "7 days across DST fall back (Nov 1 2026 is 25 h)",
			p:    Days7, now: at(2026, 11, 3, 0, 30),
			want: want{
				cur:  Range{day(2026, 10, 28), day(2026, 11, 4)},
				prev: Range{day(2026, 10, 21), day(2026, 10, 28)},
				bucketStart: []time.Time{day(2026, 10, 28), day(2026, 10, 29), day(2026, 10, 30),
					day(2026, 10, 31), day(2026, 11, 1), day(2026, 11, 2), day(2026, 11, 3)},
				lastEnd: day(2026, 11, 4),
				labels: []string{"Wed Oct 28", "Thu Oct 29", "Fri Oct 30", "Sat Oct 31",
					"Sun Nov 01", "Mon Nov 02", "Tue Nov 03"},
				curDays: 7, prevDays: 7,
			},
		},
		{
			name: "7 days across DST spring forward (Mar 8 2026 is 23 h)",
			p:    Days7, now: at(2026, 3, 9, 23, 59),
			want: want{
				cur:  Range{day(2026, 3, 3), day(2026, 3, 10)},
				prev: Range{day(2026, 2, 24), day(2026, 3, 3)},
				bucketStart: []time.Time{day(2026, 3, 3), day(2026, 3, 4), day(2026, 3, 5),
					day(2026, 3, 6), day(2026, 3, 7), day(2026, 3, 8), day(2026, 3, 9)},
				lastEnd: day(2026, 3, 10),
				labels: []string{"Tue Mar 03", "Wed Mar 04", "Thu Mar 05", "Fri Mar 06",
					"Sat Mar 07", "Sun Mar 08", "Mon Mar 09"},
				curDays: 7, prevDays: 7,
			},
		},
		{
			name: "30 days ending on a month's first day after a short February",
			p:    Days30, now: at(2026, 3, 1, 8, 0),
			want: want{
				cur:  Range{day(2026, 1, 31), day(2026, 3, 2)},
				prev: Range{day(2026, 1, 1), day(2026, 1, 31)},
				bucketStart: []time.Time{day(2026, 1, 31), day(2026, 2, 7), day(2026, 2, 14),
					day(2026, 2, 21), day(2026, 2, 28)},
				lastEnd: day(2026, 3, 2),
				labels:  []string{"Jan 31", "Feb 07", "Feb 14", "Feb 21", "Feb 28"},
				curDays: 30, prevDays: 30,
			},
		},
		{
			name: "30 days ending on a month end, across DST fall back",
			p:    Days30, now: at(2026, 11, 30, 21, 0),
			want: want{
				cur:  Range{day(2026, 11, 1), day(2026, 12, 1)},
				prev: Range{day(2026, 10, 2), day(2026, 11, 1)},
				bucketStart: []time.Time{day(2026, 11, 1), day(2026, 11, 8), day(2026, 11, 15),
					day(2026, 11, 22), day(2026, 11, 29)},
				lastEnd: day(2026, 12, 1),
				labels:  []string{"Nov 01", "Nov 08", "Nov 15", "Nov 22", "Nov 29"},
				curDays: 30, prevDays: 30,
			},
		},
		{
			name: "3 months crossing a year boundary",
			p:    Months3, now: at(2027, 1, 15, 12, 0),
			want: want{
				cur:         Range{day(2026, 11, 1), day(2027, 1, 16)},
				prev:        Range{day(2026, 8, 1), day(2026, 11, 1)},
				bucketStart: []time.Time{day(2026, 11, 1), day(2026, 12, 1), day(2027, 1, 1)},
				lastEnd:     day(2027, 1, 16),
				labels:      []string{"Nov 2026", "Dec 2026", "Jan 2027"},
				curDays:     30 + 31 + 15, prevDays: 31 + 30 + 31,
			},
		},
		{
			name: "3 months on a month end",
			p:    Months3, now: at(2026, 5, 31, 23, 59),
			want: want{
				cur:         Range{day(2026, 3, 1), day(2026, 6, 1)},
				prev:        Range{day(2025, 12, 1), day(2026, 3, 1)},
				bucketStart: []time.Time{day(2026, 3, 1), day(2026, 4, 1), day(2026, 5, 1)},
				lastEnd:     day(2026, 6, 1),
				labels:      []string{"Mar 2026", "Apr 2026", "May 2026"},
				curDays:     31 + 30 + 31, prevDays: 31 + 31 + 28,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cur, prev, buckets := Window(tc.p, tc.now)
			if !cur.Start.Equal(tc.want.cur.Start) || !cur.End.Equal(tc.want.cur.End) {
				t.Errorf("current = %v .. %v, want %v .. %v", cur.Start, cur.End, tc.want.cur.Start, tc.want.cur.End)
			}
			if !prev.Start.Equal(tc.want.prev.Start) || !prev.End.Equal(tc.want.prev.End) {
				t.Errorf("previous = %v .. %v, want %v .. %v", prev.Start, prev.End, tc.want.prev.Start, tc.want.prev.End)
			}
			if got := cur.Days(); got != tc.want.curDays {
				t.Errorf("current days = %d, want %d", got, tc.want.curDays)
			}
			if got := prev.Days(); got != tc.want.prevDays {
				t.Errorf("previous days = %d, want %d", got, tc.want.prevDays)
			}
			var starts []time.Time
			var labels []string
			for i, b := range buckets {
				starts = append(starts, b.Start)
				labels = append(labels, tc.p.Label(b.Start))
				if h, m, _ := b.Start.Clock(); h != 0 || m != 0 {
					t.Errorf("bucket %d starts at %v, not local midnight", i, b.Start)
				}
				if i > 0 && !buckets[i-1].End.Equal(b.Start) {
					t.Errorf("bucket %d does not start where %d ends", i, i-1)
				}
			}
			if !slices.EqualFunc(starts, tc.want.bucketStart, time.Time.Equal) {
				t.Errorf("bucket starts = %v, want %v", starts, tc.want.bucketStart)
			}
			if !slices.Equal(labels, tc.want.labels) {
				t.Errorf("labels = %q, want %q", labels, tc.want.labels)
			}
			if n := len(buckets); n > 0 && !buckets[n-1].End.Equal(tc.want.lastEnd) {
				t.Errorf("last bucket ends %v, want %v", buckets[n-1].End, tc.want.lastEnd)
			}
		})
	}
}

// The 30-day period's last bucket has 2 days (D-11).
func TestWindow30DaysLastBucketHasTwoDays(t *testing.T) {
	_, _, buckets := Window(Days30, at(2026, 11, 18, 9, 0))
	if len(buckets) != 5 {
		t.Fatalf("buckets = %d, want 5", len(buckets))
	}
	for i, b := range buckets {
		want := 7
		if i == 4 {
			want = 2
		}
		if got := b.Days(); got != want {
			t.Errorf("bucket %d has %d days, want %d", i, got, want)
		}
	}
}

func TestPeriodCycleAndNames(t *testing.T) {
	var got []string
	p := Days7
	for range 4 {
		got = append(got, p.String())
		p = p.Next()
	}
	if want := []string{"7 days", "30 days", "3 months", "7 days"}; !slices.Equal(got, want) {
		t.Errorf("cycle = %q, want %q", got, want)
	}
}

func completed(end time.Time, est, actual time.Duration) task.Task {
	return task.Task{
		Status:           task.Completed,
		StartedAt:        end.Add(-actual),
		EndedAt:          end,
		OriginalEstimate: est,
	}
}

func TestCompute(t *testing.T) {
	now := at(2026, 11, 3, 10, 0) // 7 days = Oct 28 .. Nov 3; previous Oct 21 .. Oct 27
	paused := completed(at(2026, 11, 2, 12, 0), 30*time.Minute, time.Hour)
	paused.Pauses = []task.Pause{{StartedAt: at(2026, 11, 2, 11, 10), EndedAt: at(2026, 11, 2, 11, 30)}}
	paused.Extensions = []task.Extension{{At: at(2026, 11, 2, 11, 40), Added: 15 * time.Minute}}

	tasks := []task.Task{
		completed(at(2026, 10, 28, 0, 0), time.Hour, 50*time.Minute),       // first instant of period
		completed(at(2026, 11, 1, 23, 30), 20*time.Minute, 25*time.Minute), // late on the 25 h day
		completed(at(2026, 11, 1, 1, 30), 10*time.Minute, 5*time.Minute),   // in the repeated hour
		paused,
		completed(at(2026, 10, 27, 23, 59), time.Hour, time.Hour), // last minute of previous
		completed(at(2026, 10, 21, 0, 0), time.Hour, time.Hour),   // first instant of previous
		completed(at(2026, 10, 20, 23, 59), time.Hour, time.Hour), // before previous: ignored
		{Status: task.Cancelled, StartedAt: at(2026, 11, 2, 9, 0), EndedAt: at(2026, 11, 2, 9, 30), OriginalEstimate: time.Hour},
		{Status: task.Active, StartedAt: at(2026, 11, 3, 9, 0), OriginalEstimate: time.Hour},
	}
	s := Compute(Days7, now, tasks)

	if s.Completed != (Comparison{4, 2}) {
		t.Errorf("completed = %+v, want {4 2}", s.Completed)
	}
	if s.PerDay != (Comparison{4.0 / 7, 2.0 / 7}) {
		t.Errorf("per day = %+v, want {4/7 2/7}", s.PerDay)
	}
	if s.Empty() {
		t.Error("Empty() = true with completed tasks")
	}
	type agg struct {
		n        int
		est, act time.Duration
	}
	want := []agg{
		{1, time.Hour, 50 * time.Minute}, // Oct 28
		{}, {}, {},                       // Oct 29 .. 31
		{2, 30 * time.Minute, 30 * time.Minute}, // Nov 1
		{1, 45 * time.Minute, 40 * time.Minute}, // Nov 2: estimate 30+15, actual 60−20
		{},                                      // Nov 3
	}
	for i, b := range s.Buckets {
		if got := (agg{b.Completed, b.Estimated, b.Actual}); got != want[i] {
			t.Errorf("bucket %d (%s) = %+v, want %+v", i, b.Label, got, want[i])
		}
	}
}

func TestComputeAttributesByLocalEndDate(t *testing.T) {
	// 23:30 local on Nov 17 is Nov 18 in UTC; the task belongs to Nov 17 (R-44).
	end := at(2026, 11, 17, 23, 30)
	tk := completed(end.UTC(), time.Hour, time.Hour)
	s := Compute(Days7, at(2026, 11, 18, 9, 0), []task.Task{tk})
	for _, b := range s.Buckets {
		wantN := 0
		if b.Label == "Tue Nov 17" {
			wantN = 1
		}
		if b.Completed != wantN {
			t.Errorf("%s completed = %d, want %d", b.Label, b.Completed, wantN)
		}
	}
}

func TestComputeThreeMonthsAveragesOverElapsedDays(t *testing.T) {
	now := at(2027, 1, 15, 12, 0) // 76 days elapsed; previous Aug..Oct = 92 days
	tasks := []task.Task{
		completed(at(2026, 11, 1, 9, 0), time.Hour, time.Hour),
		completed(at(2027, 1, 15, 11, 0), time.Hour, time.Hour),
		completed(at(2026, 8, 1, 9, 0), time.Hour, time.Hour),
	}
	s := Compute(Months3, now, tasks)
	if s.PerDay != (Comparison{2.0 / 76, 1.0 / 92}) {
		t.Errorf("per day = %+v, want {2/76 1/92}", s.PerDay)
	}
	if got := []int{s.Buckets[0].Completed, s.Buckets[1].Completed, s.Buckets[2].Completed}; !slices.Equal(got, []int{1, 0, 1}) {
		t.Errorf("bucket counts = %v, want [1 0 1]", got)
	}
}

func TestComputeEmpty(t *testing.T) {
	s := Compute(Days30, at(2026, 11, 18, 9, 0), nil)
	if !s.Empty() || len(s.Buckets) != 5 {
		t.Errorf("empty summary: Empty=%v buckets=%d", s.Empty(), len(s.Buckets))
	}
	if _, ok := s.Completed.Percent(); ok {
		t.Error("Percent ok with previous 0; want false (shown as —)")
	}
}

func TestComparison(t *testing.T) {
	c := Comparison{Current: 6, Previous: 4}
	if c.Delta() != 2 {
		t.Errorf("delta = %v, want 2", c.Delta())
	}
	if p, ok := c.Percent(); !ok || p != 50 {
		t.Errorf("percent = %v, %v; want 50, true", p, ok)
	}
	if p, ok := (Comparison{1, 4}).Percent(); !ok || p != -75 {
		t.Errorf("percent = %v, %v; want -75, true", p, ok)
	}
}
