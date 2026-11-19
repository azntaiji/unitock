package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/store/storetest"
)

// startErr runs the app expecting a startup error: exit 1, the UI never runs, and stderr
// is one line, "unitock: ..." containing every want.
func startErr(t *testing.T, dir string, want ...string) {
	t.Helper()
	stubUI(t, func(context.Context, deps) error { t.Error("UI ran"); return nil })
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != exitStartup {
		t.Errorf("exit = %d, want %d", code, exitStartup)
	}
	msg := errb.String()
	if !strings.HasPrefix(msg, "unitock: ") || strings.Count(msg, "\n") != 1 {
		t.Errorf("stderr = %q, want one line starting \"unitock: \"", msg)
	}
	for _, w := range want {
		if !strings.Contains(msg, w) {
			t.Errorf("stderr = %q, want it to mention %q", msg, w)
		}
	}
}

func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
}

// P4-03: a data dir that cannot be used gives a clear message naming it, and nothing is
// created.
func TestErrorUnwritableDataDir(t *testing.T) {
	t.Run("exists, not writable", func(t *testing.T) {
		skipIfRoot(t)
		dir := filepath.Join(t.TempDir(), "data")
		if err := os.Mkdir(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		startErr(t, dir, "data dir "+dir+" is not writable", "permission denied")
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("files created in an unwritable dir: %v", entries)
		}
	})
	t.Run("parent not writable", func(t *testing.T) {
		skipIfRoot(t)
		parent := filepath.Join(t.TempDir(), "parent")
		if err := os.Mkdir(parent, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
		dir := filepath.Join(parent, "data")
		startErr(t, dir, "data dir", dir, "permission denied")
	})
	t.Run("is a file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(file, []byte("keep me"), 0o600); err != nil {
			t.Fatal(err)
		}
		startErr(t, file, "data dir", file, "not a directory")
		if b, _ := os.ReadFile(file); string(b) != "keep me" {
			t.Errorf("the file was changed: %q", b)
		}
	})
}

// P4-03: a database the app cannot read gives a clear message naming the file, and the
// file is left exactly as it was: no data loss.
func TestErrorCorruptDatabase(t *testing.T) {
	random := make([]byte, 8192)
	for i := range random {
		random[i] = byte(i*7919 + i/13)
	}
	for _, tc := range []struct {
		name string
		body []byte
		want string
	}{
		{"random bytes", random, "file is not a database"},
		{"text", []byte("not a database, just text padding it out\n"), "file is not a database"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			db := filepath.Join(dir, store.DBFile)
			if err := os.WriteFile(db, tc.body, 0o600); err != nil {
				t.Fatal(err)
			}
			startErr(t, dir, "database "+db, tc.want)
			if b, _ := os.ReadFile(db); !bytes.Equal(b, tc.body) {
				t.Error("the unreadable database was changed")
			}
		})
	}

	t.Run("not readable", func(t *testing.T) {
		skipIfRoot(t)
		dir := t.TempDir()
		db := filepath.Join(dir, store.DBFile)
		if err := os.WriteFile(db, nil, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(db, 0o600) })
		startErr(t, dir, "database "+db, "unable to open")
	})

	// A database from a newer unitock with real history: refused, and its tasks survive.
	t.Run("newer schema", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()
		if err := storetest.Seed(ctx, dir, 50, 24*time.Hour, time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
		db := filepath.Join(dir, store.DBFile)
		raw, err := sql.Open("sqlite", "file:"+db)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := raw.ExecContext(ctx, "PRAGMA user_version = 999"); err != nil {
			t.Fatal(err)
		}
		_ = raw.Close()
		before, err := os.ReadFile(db)
		if err != nil {
			t.Fatal(err)
		}
		startErr(t, dir, "database "+db, "written by a newer unitock")
		if after, _ := os.ReadFile(db); !bytes.Equal(before, after) {
			t.Error("the newer database was changed")
		}
		raw, err = sql.Open("sqlite", "file:"+db)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = raw.Close() }()
		var n int
		if err := raw.QueryRowContext(ctx, "SELECT count(*) FROM tasks").Scan(&n); err != nil || n != 50 {
			t.Errorf("tasks after the refused start: %d, %v; want 50", n, err)
		}
	})
}
