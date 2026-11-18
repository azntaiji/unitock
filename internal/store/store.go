// Package store persists tasks in SQLite (D-05, D-07). The schema is created and
// upgraded by numbered SQL migrations embedded in the binary, applied in one
// transaction at Open and tracked with PRAGMA user_version.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/azntaiji/unitock/internal/task"
)

// DBFile is the database file name inside the data directory.
const DBFile = "unitock.db"

// Errors returned by Store methods.
var (
	ErrActiveExists = errors.New("another task is already active")
	ErrNoActive     = errors.New("no active task")
	ErrNotActive    = errors.New("task is not active")
	ErrNotFound     = errors.New("task not found")
	ErrPaused       = errors.New("task is already paused")
	ErrNotPaused    = errors.New("task is not paused")
	ErrNewerSchema  = errors.New("database was written by a newer unitock")
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Store is a SQLite-backed task store. It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database in dir and applies pending migrations.
// dir is created with mode 0700 if it does not exist.
func Open(ctx context.Context, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	q := url.Values{}
	for _, p := range []string{"journal_mode(WAL)", "foreign_keys(1)", "busy_timeout(5000)", "synchronous(NORMAL)"} {
		q.Add("_pragma", p)
	}
	dsn := "file:" + filepath.ToSlash(filepath.Join(dir, DBFile)) + "?" + q.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	// One connection: SQLite serialises writes anyway, and this keeps transactions and
	// per-connection pragmas simple.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// SchemaVersion returns the applied migration number.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	var v int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("reading schema version: %w", err)
	}
	return v, nil
}

type migration struct {
	version int
	name    string
}

func migrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading migrations: %w", err)
	}
	var out []migration
	for _, e := range entries {
		num, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(num)
		if !ok || err != nil || v <= 0 {
			return nil, fmt.Errorf("bad migration file name %q", e.Name())
		}
		out = append(out, migration{version: v, name: e.Name()})
	}
	slices.SortFunc(out, func(a, b migration) int { return a.version - b.version })
	for i, m := range out {
		if m.version != i+1 {
			return nil, fmt.Errorf("migrations not numbered 1..n: %q", m.name)
		}
	}
	return out, nil
}

func (s *Store) migrate(ctx context.Context) error {
	ms, err := migrations()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var current int
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	if current > len(ms) {
		return fmt.Errorf("%w: schema %d, this build knows %d", ErrNewerSchema, current, len(ms))
	}
	for _, m := range ms[current:] {
		body, err := migrationFS.ReadFile("migrations/" + m.name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("applying migration %s: %w", m.name, err)
		}
		// PRAGMA does not take bound parameters; the value is an integer we control.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			return fmt.Errorf("recording migration %s: %w", m.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migrations: %w", err)
	}
	return nil
}

// Create stores a new active task and returns it. It returns ErrActiveExists if a task
// is already active (R-03).
func (s *Store) Create(ctx context.Context, name string, estimate time.Duration, startedAt time.Time) (task.Task, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (name, status, started_at, original_estimate_ms) VALUES (?, ?, ?, ?)`,
		name, string(task.Active), startedAt.UnixMilli(), estimate.Milliseconds())
	if err != nil {
		if isUniqueViolation(err) {
			return task.Task{}, ErrActiveExists
		}
		return task.Task{}, fmt.Errorf("creating task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return task.Task{}, fmt.Errorf("creating task: %w", err)
	}
	return s.Get(ctx, id)
}

// Active returns the active task, or ErrNoActive.
func (s *Store) Active(ctx context.Context) (task.Task, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM tasks WHERE status = ?`, string(task.Active)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, ErrNoActive
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("finding active task: %w", err)
	}
	return s.Get(ctx, id)
}

// Get returns the task with the given id, with its extensions and pauses.
func (s *Store) Get(ctx context.Context, id int64) (task.Task, error) {
	var (
		t       task.Task
		status  string
		started int64
		ended   sql.NullInt64
		estMS   int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, status, started_at, ended_at, original_estimate_ms FROM tasks WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &status, &started, &ended, &estMS)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, ErrNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("reading task %d: %w", id, err)
	}
	t.Status = task.Status(status)
	t.StartedAt = fromMS(started)
	if ended.Valid {
		t.EndedAt = fromMS(ended.Int64)
	}
	t.OriginalEstimate = time.Duration(estMS) * time.Millisecond

	if t.Extensions, err = s.extensions(ctx, id); err != nil {
		return task.Task{}, err
	}
	if t.Pauses, err = s.pauses(ctx, id); err != nil {
		return task.Task{}, err
	}
	return t, nil
}

func (s *Store) extensions(ctx context.Context, id int64) ([]task.Extension, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT at, added_ms FROM extensions WHERE task_id = ? ORDER BY at, id`, id)
	if err != nil {
		return nil, fmt.Errorf("reading extensions of task %d: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	var out []task.Extension
	for rows.Next() {
		var at, added int64
		if err := rows.Scan(&at, &added); err != nil {
			return nil, fmt.Errorf("reading extensions of task %d: %w", id, err)
		}
		out = append(out, task.Extension{At: fromMS(at), Added: time.Duration(added) * time.Millisecond})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading extensions of task %d: %w", id, err)
	}
	return out, nil
}

func (s *Store) pauses(ctx context.Context, id int64) ([]task.Pause, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT started_at, ended_at FROM pauses WHERE task_id = ? ORDER BY started_at, id`, id)
	if err != nil {
		return nil, fmt.Errorf("reading pauses of task %d: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	var out []task.Pause
	for rows.Next() {
		var started int64
		var ended sql.NullInt64
		if err := rows.Scan(&started, &ended); err != nil {
			return nil, fmt.Errorf("reading pauses of task %d: %w", id, err)
		}
		p := task.Pause{StartedAt: fromMS(started)}
		if ended.Valid {
			p.EndedAt = fromMS(ended.Int64)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading pauses of task %d: %w", id, err)
	}
	return out, nil
}

// Complete marks an active task completed at endedAt, closing any open pause at the
// same instant. It returns ErrNotActive if the task is not active.
func (s *Store) Complete(ctx context.Context, id int64, endedAt time.Time) error {
	return s.end(ctx, id, task.Completed, endedAt)
}

// Pause opens a pause on an active, running task at at and returns the updated task
// (R-20). It returns ErrNotActive or ErrPaused.
func (s *Store) Pause(ctx context.Context, id int64, at time.Time) (task.Task, error) {
	return s.setPaused(ctx, id, at, true)
}

// Resume closes the open pause of an active task at at and returns the updated task
// (R-20). It returns ErrNotActive or ErrNotPaused.
func (s *Store) Resume(ctx context.Context, id int64, at time.Time) (task.Task, error) {
	return s.setPaused(ctx, id, at, false)
}

func (s *Store) setPaused(ctx context.Context, id int64, at time.Time, pause bool) (task.Task, error) {
	verb := "resuming"
	if pause {
		verb = "pausing"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return task.Task{}, fmt.Errorf("%s task %d: %w", verb, id, err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var open int
	err = tx.QueryRowContext(ctx,
		`SELECT status, (SELECT count(*) FROM pauses WHERE task_id = tasks.id AND ended_at IS NULL)
		 FROM tasks WHERE id = ?`, id).Scan(&status, &open)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return task.Task{}, ErrNotFound
	case err != nil:
		return task.Task{}, fmt.Errorf("%s task %d: %w", verb, id, err)
	case status != string(task.Active):
		return task.Task{}, ErrNotActive
	case pause && open > 0:
		return task.Task{}, ErrPaused
	case !pause && open == 0:
		return task.Task{}, ErrNotPaused
	}
	ms := at.UnixMilli()
	if pause {
		_, err = tx.ExecContext(ctx, `INSERT INTO pauses (task_id, started_at) VALUES (?, ?)`, id, ms)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE pauses SET ended_at = ? WHERE task_id = ? AND ended_at IS NULL`, ms, id)
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("%s task %d: %w", verb, id, err)
	}
	if err := tx.Commit(); err != nil {
		return task.Task{}, fmt.Errorf("%s task %d: %w", verb, id, err)
	}
	return s.Get(ctx, id)
}

func (s *Store) end(ctx context.Context, id int64, status task.Status, endedAt time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ending task %d: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()

	ms := endedAt.UnixMilli()
	res, err := tx.ExecContext(ctx,
		`UPDATE tasks SET status = ?, ended_at = ? WHERE id = ? AND status = ?`,
		string(status), ms, id, string(task.Active))
	if err != nil {
		return fmt.Errorf("ending task %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ending task %d: %w", id, err)
	}
	if n == 0 {
		return ErrNotActive
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE pauses SET ended_at = ? WHERE task_id = ? AND ended_at IS NULL`, ms, id); err != nil {
		return fmt.Errorf("closing pauses of task %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ending task %d: %w", id, err)
	}
	return nil
}

func fromMS(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	return errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}
