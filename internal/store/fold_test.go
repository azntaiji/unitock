package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// foldKey must agree with strings.EqualFold on every pair (amended D-15).
func TestFoldKeyMatchesEqualFold(t *testing.T) {
	names := []string{
		"über", "Über", "ÜBER", "uber", "unitock", "UNITOCK", "UniTock",
		"K", "k", "K", // Kelvin sign folds with K
		"ß", "ss", "SS", "ẞ",
		"Σίσυφος", "σίσυφος", "ΣΊΣΥΦΟΣ", "ς",
		"İstanbul", "istanbul", "ıstanbul",
		"Ǆ", "ǅ", "ǆ", "", "日本", "a b", "a  b",
	}
	for _, a := range names {
		for _, b := range names {
			want := strings.EqualFold(a, b)
			if got := foldKey(a) == foldKey(b); got != want {
				t.Errorf("foldKey(%q)==foldKey(%q) is %v; EqualFold is %v", a, b, got, want)
			}
		}
	}
}

func TestProjectCreateReusesUnicodeCaseMatch(t *testing.T) {
	ctx := context.Background()
	s, _ := openTemp(t)
	first, err := s.Create(ctx, "Über", "a", time.Hour, t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(ctx, first.ID, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(ctx, "über", "b", time.Hour, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.Project != "Über" {
		t.Errorf("project = %q, want the stored spelling Über", second.Project)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&n); err != nil || n != 1 {
		t.Errorf("projects rows = %d (%v), want 1", n, err)
	}
}

// A v0.2.0 database (schema 2) holding "über" and "Über" as two projects upgrades to
// one project, the older spelling, owning both tasks.
func TestProjectKeyMigrationMergesDuplicates(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, DBFile))
	if err != nil {
		t.Fatal(err)
	}
	var stmts []string
	for _, f := range []string{"0001_init.sql", "0002_projects.sql"} {
		b, err := migrationFS.ReadFile("migrations/" + f)
		if err != nil {
			t.Fatal(err)
		}
		stmts = append(stmts, string(b))
	}
	stmts = append(stmts, "PRAGMA user_version = 2",
		`INSERT INTO projects (id, name, created_at) VALUES (1, 'über', 2000), (2, 'Über', 1000), (3, 'other', 1500)`,
		`INSERT INTO tasks (name, status, started_at, ended_at, original_estimate_ms, project_id) VALUES
			('a', 'completed', 1000, 2000, 60000, 2),
			('b', 'completed', 3000, 4000, 60000, 1),
			('c', 'completed', 5000, 6000, 60000, 3)`)
	for _, q := range stmts {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()

	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	for id, want := range map[int64]string{1: "Über", 2: "Über", 3: "other"} {
		got, err := s.Get(ctx, id)
		if err != nil || got.Project != want {
			t.Errorf("task %d project = %q (%v), want %q", id, got.Project, err, want)
		}
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&n); err != nil || n != 2 {
		t.Errorf("projects rows = %d (%v), want 2", n, err)
	}
	// New starts reuse the merged project, and the key index rejects a duplicate.
	tk, err := s.Create(ctx, "ÜBER", "d", time.Hour, t0)
	if err != nil || tk.Project != "Über" {
		t.Errorf("Create ÜBER = %q, %v; want Über", tk.Project, err)
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, name_key, created_at) VALUES ('x', ?, 0)`, foldKey("über")); err == nil {
		t.Error("duplicate name_key accepted; want unique index violation")
	}
}
