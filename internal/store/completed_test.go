package store

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// Completed returns only completed tasks that ended in [from, to), each equal to what
// Get returns for it (extensions and pauses included).
func TestCompletedRangeAndDetail(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	from, to := t0, t0.Add(24*time.Hour)

	mk := func(project string, start time.Time, run time.Duration, end func(id int64, at time.Time) error) int64 {
		t.Helper()
		tk, err := s.Create(ctx, project, "x", time.Hour, start)
		if err != nil {
			t.Fatal(err)
		}
		if end != nil {
			if err := end(tk.ID, start.Add(run)); err != nil {
				t.Fatal(err)
			}
		}
		return tk.ID
	}
	complete := func(id int64, at time.Time) error { return s.Complete(ctx, id, at) }
	cancel := func(id int64, at time.Time) error { return s.Cancel(ctx, id, at) }

	before := mk("", from.Add(-2*time.Hour), 2*time.Hour, complete) // ends exactly at from: in
	// one with an extension and a pause
	detailed, err := s.Create(ctx, "unitock", "detailed", 30*time.Minute, from.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pause(ctx, detailed.ID, from.Add(70*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resume(ctx, detailed.ID, from.Add(80*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddExtension(ctx, detailed.ID, from.Add(85*time.Minute), 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(ctx, detailed.ID, from.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	mk("", from.Add(3*time.Hour), time.Hour, cancel)               // cancelled: out
	mk("", to.Add(-time.Hour), time.Hour, complete)                // ends exactly at to: out
	early := mk("", from.Add(-3*time.Hour), time.Minute, complete) // ends before from: out
	mk("", to.Add(time.Hour), 0, nil)                              // still active: out

	got, err := s.Completed(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, tk := range got {
		ids = append(ids, tk.ID)
		want, err := s.Get(ctx, tk.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tk, want) {
			t.Errorf("Completed task %d = %+v\nGet = %+v", tk.ID, tk, want)
		}
	}
	if want := []int64{before, detailed.ID}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v, want %v (early %d excluded)", ids, want, early)
	}
	if len(got) == 2 && (len(got[1].Pauses) != 1 || len(got[1].Extensions) != 1) {
		t.Errorf("detailed task pauses/extensions = %d/%d, want 1/1", len(got[1].Pauses), len(got[1].Extensions))
	}

	none, err := s.Completed(ctx, to.Add(48*time.Hour), to.Add(72*time.Hour))
	if err != nil || len(none) != 0 {
		t.Errorf("empty range = %v, %v", none, err)
	}
}
