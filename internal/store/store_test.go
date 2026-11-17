package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/task"
)

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	s, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, dir
}

func TestOpenMigratesFreshDataDir(t *testing.T) {
	ctx := context.Background()
	s, dir := openTemp(t)

	if _, err := os.Stat(filepath.Join(dir, DBFile)); err != nil {
		t.Fatalf("database file not created: %v", err)
	}
	ms, err := migrations()
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v != len(ms) || v < 1 {
		t.Errorf("schema version = %d, want %d", v, len(ms))
	}
	for _, table := range []string{"tasks", "extensions", "pauses"} {
		var n int
		if err := s.db.QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("table %s missing", table)
		}
	}
	var mode string
	if err := s.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestReopenIsIdempotentAndKeepsData(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(ctx, "write report", 30*time.Minute, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s.Close() }()
	got, err := s.Active(ctx)
	if err != nil {
		t.Fatalf("Active after reopen: %v", err)
	}
	if got.ID != created.ID || got.Name != "write report" {
		t.Errorf("Active after reopen = %+v, want %+v", got, created)
	}
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	ctx := context.Background()
	s, dir := openTemp(t)
	if _, err := s.db.ExecContext(ctx, "PRAGMA user_version = 999"); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	if _, err := Open(ctx, dir); !errors.Is(err, ErrNewerSchema) {
		t.Fatalf("Open = %v, want ErrNewerSchema", err)
	}
}

func TestCreatePersistsStartedTask(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	started := t0.Add(1234 * time.Millisecond)

	got, err := s.Create(ctx, "Deep work", 90*time.Minute, started)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	want := task.Task{
		ID: got.ID, Name: "Deep work", Status: task.Active,
		StartedAt: started, OriginalEstimate: 90 * time.Minute,
	}
	if got.ID == 0 || got.Name != want.Name || got.Status != want.Status ||
		!got.StartedAt.Equal(want.StartedAt) || !got.EndedAt.IsZero() ||
		got.OriginalEstimate != want.OriginalEstimate || len(got.Extensions) != 0 || len(got.Pauses) != 0 {
		t.Errorf("Create = %+v, want %+v", got, want)
	}

	// The stored row, as written.
	var (
		name, status     string
		startedMS, estMS int64
		endedNull        bool
	)
	if err := s.db.QueryRowContext(ctx,
		`SELECT name, status, started_at, ended_at IS NULL, original_estimate_ms FROM tasks WHERE id = ?`, got.ID).
		Scan(&name, &status, &startedMS, &endedNull, &estMS); err != nil {
		t.Fatal(err)
	}
	if name != "Deep work" || status != "active" || startedMS != started.UnixMilli() || !endedNull || estMS != 90*60*1000 {
		t.Errorf("row = %q %q %d null=%v %d", name, status, startedMS, endedNull, estMS)
	}

	active, err := s.Active(ctx)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if active.ID != got.ID {
		t.Errorf("Active ID = %d, want %d", active.ID, got.ID)
	}
}

func TestSecondActiveTaskRejected(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	first, err := s.Create(ctx, "one", time.Hour, t0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "two", time.Hour, t0.Add(time.Minute)); !errors.Is(err, ErrActiveExists) {
		t.Fatalf("second Create = %v, want ErrActiveExists", err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM tasks`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("tasks rows = %d, want 1", n)
	}

	// Once the first is completed a new task can start.
	if err := s.Complete(ctx, first.ID, t0.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "two", time.Hour, t0.Add(31*time.Minute)); err != nil {
		t.Fatalf("Create after Complete: %v", err)
	}
}

func TestCompleteMarksTaskCompleted(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	tk, err := s.Create(ctx, "focus", 25*time.Minute, t0)
	if err != nil {
		t.Fatal(err)
	}
	// An open pause must be closed at completion.
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO pauses (task_id, started_at) VALUES (?, ?)`, tk.ID, t0.Add(10*time.Minute).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	end := t0.Add(28 * time.Minute)
	if err := s.Complete(ctx, tk.ID, end); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	got, err := s.Get(ctx, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != task.Completed || !got.EndedAt.Equal(end) {
		t.Errorf("after Complete: status %q ended %v", got.Status, got.EndedAt)
	}
	if len(got.Pauses) != 1 || !got.Pauses[0].EndedAt.Equal(end) {
		t.Errorf("pauses = %+v, want one ended at %v", got.Pauses, end)
	}
	if got.Elapsed(end.Add(time.Hour)) != 10*time.Minute {
		t.Errorf("Elapsed = %v, want 10m", got.Elapsed(end.Add(time.Hour)))
	}
	if _, err := s.Active(ctx); !errors.Is(err, ErrNoActive) {
		t.Errorf("Active = %v, want ErrNoActive", err)
	}
	if err := s.Complete(ctx, tk.ID, end); !errors.Is(err, ErrNotActive) {
		t.Errorf("second Complete = %v, want ErrNotActive", err)
	}
}

func TestGetLoadsExtensionsAndPauses(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	tk, err := s.Create(ctx, "x", 30*time.Minute, t0)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO extensions (task_id, at, added_ms) VALUES (?, ?, ?)`, []any{tk.ID, t0.Add(30 * time.Minute).UnixMilli(), 600000}},
		{`INSERT INTO pauses (task_id, started_at, ended_at) VALUES (?, ?, ?)`, []any{tk.ID, t0.Add(5 * time.Minute).UnixMilli(), t0.Add(7 * time.Minute).UnixMilli()}},
		{`INSERT INTO pauses (task_id, started_at) VALUES (?, ?)`, []any{tk.ID, t0.Add(20 * time.Minute).UnixMilli()}},
	} {
		if _, err := s.db.ExecContext(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Get(ctx, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Estimate() != 40*time.Minute {
		t.Errorf("Estimate = %v, want 40m", got.Estimate())
	}
	if len(got.Pauses) != 2 || !got.Paused() {
		t.Errorf("pauses = %+v, want two with the last open", got.Pauses)
	}
	if got.Elapsed(t0.Add(time.Hour)) != 18*time.Minute {
		t.Errorf("Elapsed = %v, want 18m", got.Elapsed(t0.Add(time.Hour)))
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	if _, err := s.Active(ctx); !errors.Is(err, ErrNoActive) {
		t.Errorf("Active on empty store = %v, want ErrNoActive", err)
	}
	if _, err := s.Get(ctx, 42); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(42) = %v, want ErrNotFound", err)
	}
	if err := s.Complete(ctx, 42, t0); !errors.Is(err, ErrNotActive) {
		t.Errorf("Complete(42) = %v, want ErrNotActive", err)
	}
	if _, err := s.Create(ctx, "", time.Hour, t0); err == nil {
		t.Error("Create with empty name succeeded")
	}
	if _, err := s.Create(ctx, "x", 0, t0); err == nil {
		t.Error("Create with zero estimate succeeded")
	}
}
