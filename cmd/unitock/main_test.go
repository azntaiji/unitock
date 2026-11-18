package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azntaiji/unitock/internal/store"
)

func noEnv(string) string { return "" }

func stubUI(t *testing.T, f func(context.Context, deps) error) {
	t.Helper()
	old := runUI
	runUI = f
	t.Cleanup(func() { runUI = old })
}

func TestVersionFlag(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"--version"}, &out, &errb, noEnv)
	if code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errb.String())
	}
	if got, want := out.String(), "unitock dev (none, unknown)\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestBadArgumentsExitOne(t *testing.T) {
	for _, args := range [][]string{{"--nope"}, {"extra"}, {"--data-dir"}} {
		var out, errb bytes.Buffer
		if code := run(context.Background(), args, &out, &errb, noEnv); code != 1 {
			t.Errorf("run(%q) = %d, want 1", args, code)
		}
		if !strings.Contains(errb.String(), "usage: unitock") {
			t.Errorf("run(%q) stderr lacks usage: %q", args, errb.String())
		}
	}
}

func TestHelpExitsZero(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"-h"}, &out, &errb, noEnv); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestStartCreatesDataDirLogAndDatabase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")
	var got deps
	stubUI(t, func(_ context.Context, d deps) error { got = d; return nil })

	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errb.String())
	}
	if got.store == nil || got.clock == nil || got.log == nil {
		t.Fatalf("UI got incomplete deps: %+v", got)
	}
	fi, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("data dir mode = %o, want 700", perm)
	}
	if _, err := os.Stat(filepath.Join(dir, store.DBFile)); err != nil {
		t.Errorf("database missing: %v", err)
	}
	logBody, err := os.ReadFile(filepath.Join(dir, LogFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logBody), "msg=start") || !strings.Contains(string(logBody), "msg=quit") {
		t.Errorf("log = %q, want start and quit lines", logBody)
	}
}

func TestUnusableDataDirExitsOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stubUI(t, func(context.Context, deps) error { t.Error("UI ran"); return nil })
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", file}, &out, &errb, noEnv); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.HasPrefix(errb.String(), "unitock: ") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUnusableDatabaseExitsOne(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, store.DBFile), []byte("not a database, just text padding it out"), 0o600); err != nil {
		t.Fatal(err)
	}
	stubUI(t, func(context.Context, deps) error { t.Error("UI ran"); return nil })
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "database") {
		t.Errorf("stderr = %q, want a database error", errb.String())
	}
}

func TestUIErrorExitsOne(t *testing.T) {
	stubUI(t, func(context.Context, deps) error { return errors.New("boom") })
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", t.TempDir()}, &out, &errb, noEnv); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
}

func TestDefaultDataDir(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{"xdg set", map[string]string{"XDG_DATA_HOME": "/x/data", "HOME": "/home/u"}, "/x/data/unitock", false},
		{"xdg unset", map[string]string{"HOME": "/home/u"}, "/home/u/.local/share/unitock", false},
		{"xdg relative is ignored", map[string]string{"XDG_DATA_HOME": "rel", "HOME": "/home/u"}, "/home/u/.local/share/unitock", false},
		{"nothing set", map[string]string{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := defaultDataDir(func(k string) string { return tt.env[k] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// A second instance on the same data directory exits 1 naming the lock file (R-09).
func TestLockSecondInstanceExitsOne(t *testing.T) {
	dir := t.TempDir()
	stubUI(t, func(_ context.Context, d deps) error {
		var out, errb bytes.Buffer
		if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != 1 {
			t.Errorf("second instance exit = %d, want 1", code)
		}
		lock := filepath.Join(dir, LockFile)
		if msg := errb.String(); !strings.Contains(msg, lock) || !strings.Contains(msg, "already running") {
			t.Errorf("stderr %q, want it to name %s", msg, lock)
		}
		return nil
	})
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != 0 {
		t.Fatalf("first instance exit = %d, stderr %q", code, errb.String())
	}
	// Released at exit: a later run starts normally.
	stubUI(t, func(context.Context, deps) error { return nil })
	if code := run(context.Background(), []string{"--data-dir", dir}, &out, &errb, noEnv); code != 0 {
		t.Errorf("run after release exit = %d, stderr %q", code, errb.String())
	}
}

// Other data directories are independent.
func TestLockIsPerDataDir(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	release, err := acquireLock(a)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := acquireLock(a); !errors.Is(err, errLocked) {
		t.Errorf("same dir: err %v, want errLocked", err)
	}
	other, err := acquireLock(b)
	if err != nil {
		t.Fatalf("other dir: %v", err)
	}
	other()
}
