// Package storetest seeds large task histories for tests, such as the performance
// checks (SPEC Constraints, P4-02). It writes rows directly in one transaction, because
// seeding 10,000 tasks through the store API takes tens of thousands of commits.
package storetest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // the "sqlite" driver; store imports it too

	"github.com/azntaiji/unitock/internal/store"
)

// Projects is how many projects Seed spreads tasks across.
const Projects = 20

// Seed creates the database in dir (migrated by store.Open) and adds n ended tasks that
// started within span before now: about one in 20 cancelled and the rest completed,
// every fourth with an extension, every fifth with a pause, and all but every third in
// one of Projects projects. There is no active task. The history is fixed for a given
// n, span, and now.
func Seed(ctx context.Context, dir string, n int, span time.Duration, now time.Time) error {
	st, err := store.Open(ctx, dir)
	if err != nil {
		return err
	}
	if err := st.Close(); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, store.DBFile))+"?_pragma=foreign_keys(1)")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	origin := now.Add(-span)
	for p := 1; p <= Projects; p++ {
		name := fmt.Sprintf("Project %02d", p)
		// The store matches names by a Unicode case-folding key; for ASCII it is the
		// upper case.
		if _, err := tx.ExecContext(ctx, `INSERT INTO projects (id, name, name_key, created_at) VALUES (?, ?, ?, ?)`,
			p, name, strings.ToUpper(name), origin.UnixMilli()); err != nil {
			return fmt.Errorf("seeding project: %w", err)
		}
	}

	insTask, err := tx.PrepareContext(ctx, `INSERT INTO tasks (id, name, status, started_at, ended_at, original_estimate_ms, project_id) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	insExt, err := tx.PrepareContext(ctx, `INSERT INTO extensions (task_id, at, added_ms) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	insPause, err := tx.PrepareContext(ctx, `INSERT INTO pauses (task_id, started_at, ended_at) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}

	step := span / time.Duration(n+1)
	for i := 1; i <= n; i++ {
		start := origin.Add(time.Duration(i) * step)
		est := time.Duration(5+i%55) * time.Minute
		actual := est * time.Duration(70+i%60) / 100
		status := "completed"
		if i%20 == 0 {
			status = "cancelled"
		}
		var project sql.NullInt64
		if i%3 != 0 {
			project = sql.NullInt64{Int64: int64(1 + i%Projects), Valid: true}
		}
		end := start.Add(actual)
		if i%5 == 0 {
			end = end.Add(2 * time.Minute) // the pause below
		}
		if _, err := insTask.ExecContext(ctx, i, fmt.Sprintf("Task %d", i), status,
			start.UnixMilli(), end.UnixMilli(), est.Milliseconds(), project); err != nil {
			return fmt.Errorf("seeding task: %w", err)
		}
		if i%4 == 0 {
			if _, err := insExt.ExecContext(ctx, i, start.Add(est).UnixMilli(), (10 * time.Minute).Milliseconds()); err != nil {
				return fmt.Errorf("seeding extension: %w", err)
			}
		}
		if i%5 == 0 {
			p := start.Add(time.Minute)
			if _, err := insPause.ExecContext(ctx, i, p.UnixMilli(), p.Add(2*time.Minute).UnixMilli()); err != nil {
				return fmt.Errorf("seeding pause: %w", err)
			}
		}
	}
	return tx.Commit()
}
