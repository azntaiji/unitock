// Command unitock is a single-task terminal time tracker.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
	"github.com/azntaiji/unitock/internal/tui"
)

// Set at build time with -ldflags "-X main.version=...".
var version, commit, date = "dev", "none", "unknown"

// LogFile is the diagnostic log's name inside the data directory.
const LogFile = "unitock.log"

// Exit codes (SPEC Interfaces).
const (
	exitOK      = 0
	exitStartup = 1
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

// deps is what the UI needs from startup.
type deps struct {
	store *store.Store
	clock task.Clock
	log   *slog.Logger
}

// runUI runs the terminal UI until the user quits. Tests replace it.
var runUI = func(ctx context.Context, d deps) error {
	m, err := tui.New(tui.Deps{Ctx: ctx, Store: d.store, Clock: d.clock, Log: d.log})
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithContext(ctx)).Run()
	return err
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	fs := flag.NewFlagSet("unitock", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dataDir := fs.String("data-dir", "", "data directory (default $XDG_DATA_HOME/unitock or ~/.local/share/unitock)")
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(fs.Output(), "usage: unitock [--data-dir DIR] [--version]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitStartup
	}
	if fs.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "unitock: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return exitStartup
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, versionLine())
		return exitOK
	}

	if err := start(ctx, *dataDir, getenv); err != nil {
		_, _ = fmt.Fprintf(stderr, "unitock: %v\n", err)
		return exitStartup
	}
	return exitOK
}

func versionLine() string {
	return fmt.Sprintf("unitock %s (%s, %s)", version, commit, date)
}

// start prepares the data directory, log, and store, then runs the UI.
func start(ctx context.Context, dataDir string, getenv func(string) string) error {
	if dataDir == "" {
		d, err := defaultDataDir(getenv)
		if err != nil {
			return err
		}
		dataDir = d
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	logf, err := os.OpenFile(filepath.Join(dataDir, LogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening log: %w", err)
	}
	defer func() { _ = logf.Close() }()
	log := slog.New(slog.NewTextHandler(logf, nil))
	log.Info("start", "version", version, "commit", commit, "data_dir", dataDir)

	st, err := store.Open(ctx, dataDir)
	if err != nil {
		log.Error("open store", "err", err)
		return fmt.Errorf("database: %w", err)
	}
	defer func() { _ = st.Close() }()

	if err := runUI(ctx, deps{store: st, clock: task.SystemClock{}, log: log}); err != nil {
		log.Error("ui", "err", err)
		return err
	}
	log.Info("quit")
	return nil
}

// defaultDataDir is $XDG_DATA_HOME/unitock, or ~/.local/share/unitock when
// XDG_DATA_HOME is unset (or not absolute, per the XDG spec), on every OS.
func defaultDataDir(getenv func(string) string) (string, error) {
	if x := getenv("XDG_DATA_HOME"); x != "" && filepath.IsAbs(x) {
		return filepath.Join(x, "unitock"), nil
	}
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("cannot find a data dir: set --data-dir, XDG_DATA_HOME, or HOME")
	}
	return filepath.Join(home, ".local", "share", "unitock"), nil
}
