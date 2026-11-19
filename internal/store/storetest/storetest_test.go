package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

func TestSeedReadsBackThroughTheStore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	if err := Seed(ctx, dir, 100, 10*24*time.Hour, now); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	all, err := st.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 100 {
		t.Fatalf("All: %d tasks, want 100", len(all))
	}
	var cancelled, exts, pauses int
	for _, tk := range all {
		if tk.Status == task.Cancelled {
			cancelled++
		}
		exts += len(tk.Extensions)
		pauses += len(tk.Pauses)
		if tk.StartedAt.Before(now.Add(-10*24*time.Hour)) || !tk.EndedAt.After(tk.StartedAt) {
			t.Fatalf("task %d: started %v, ended %v", tk.ID, tk.StartedAt, tk.EndedAt)
		}
	}
	if cancelled != 5 || exts != 25 || pauses != 20 {
		t.Errorf("cancelled, extensions, pauses = %d, %d, %d; want 5, 25, 20", cancelled, exts, pauses)
	}
	done, err := st.Completed(ctx, now.Add(-30*24*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 95 {
		t.Errorf("Completed: %d tasks, want 95", len(done))
	}
	if _, err := st.Active(ctx); !errors.Is(err, store.ErrNoActive) {
		t.Errorf("Active: %v, want ErrNoActive", err)
	}

	// Seeded project keys match the store's own, so a differently cased name reuses one.
	ps, err := st.Projects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != Projects {
		t.Fatalf("Projects: %d, want %d", len(ps), Projects)
	}
	if _, err := st.Create(ctx, "project 01", "x", time.Minute, now); err != nil {
		t.Fatal(err)
	}
	if ps, _ = st.Projects(ctx); len(ps) != Projects {
		t.Errorf("after creating in \"project 01\": %d projects, want %d", len(ps), Projects)
	}
}
