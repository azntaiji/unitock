package tui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/azntaiji/unitock/internal/export"
	"github.com/azntaiji/unitock/internal/task"
)

// exportHarness opens Stats from Start with exports going to a temp dir, after one
// completed and one active task.
func exportHarness(t *testing.T) (*harness, string) {
	t.Helper()
	h := newHarness(t)
	dir := t.TempDir()
	h.m.deps.ExportDir = dir
	h.m.deps.Version = "0.3.0-test"
	h.startTask("first", "25m")
	h.clock.Advance(10 * time.Minute)
	h.press("e") // complete
	h.press("x") // any key: Done → Start
	h.startTask("second", "5m")
	h.pressStats()
	if h.m.screen != screenStats {
		t.Fatal("Stats did not open from Active")
	}
	return h, dir
}

// P3-05: e on Stats writes unitock-export-YYYYMMDD-HHMMSS.json with every task in
// schema v1 and shows the path (R-47, R-50).
func TestStatsExportWritesFileAndShowsPath(t *testing.T) {
	h, dir := exportHarness(t)
	h.press("e")
	path := filepath.Join(dir, export.FileName(h.clock.Now()))
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Exported to") || !strings.Contains(v, ".json") {
		t.Errorf("path not shown:\n%s", v)
	}
	if h.m.screen != screenStats || h.taskCount() != 1 {
		t.Errorf("e on Stats changed screen (%d) or the active task", h.m.screen)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc export.Document
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || doc.AppVersion != "0.3.0-test" || len(doc.Tasks) != 2 ||
		doc.Tasks[0].Status != "completed" || doc.Tasks[1].Status != "active" {
		t.Errorf("export = %+v", doc)
	}
	if !strings.Contains(h.view(), "e export") {
		t.Errorf("hints lack e export:\n%s", h.view())
	}
}

// P3-05: a second export in the same second does not overwrite; the error is shown.
func TestStatsExportNeverOverwritesShowsError(t *testing.T) {
	h, dir := exportHarness(t)
	h.press("e")
	path := filepath.Join(dir, export.FileName(h.clock.Now()))
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h.press("e")
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Export failed") || !strings.Contains(v, "already exists") {
		t.Errorf("overwrite error not shown:\n%s", v)
	}
	if again, _ := os.ReadFile(path); string(again) != string(first) {
		t.Error("existing export was changed")
	}
	// A second later there is a new name, and it succeeds.
	h.clock.Advance(time.Second)
	h.press("e")
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Exported to") {
		t.Errorf("later export failed:\n%s", v)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Errorf("export dir has %d files, want 2", len(entries))
	}
}

// allErrStore is a Store whose All fails.
type allErrStore struct{ Store }

func (allErrStore) All(context.Context) ([]task.Task, error) { return nil, errDisk }

// A store error is shown, not swallowed.
func TestStatsExportStoreErrorShown(t *testing.T) {
	h, _ := exportHarness(t)
	h.m.deps.Store = allErrStore{h.m.deps.Store}
	h.press("e")
	if v := ansi.Strip(h.view()); !strings.Contains(v, "Export failed: disk on fire") {
		t.Errorf("store error not shown:\n%s", v)
	}
}

// A long path keeps its label and file name; a long error keeps its start. Both fit.
func TestStatsExportNoteFitsWidth(t *testing.T) {
	long := "/" + strings.Repeat("very/", 30) + "unitock-export-20261117-090000.json"
	m := statsModel{note: exportedPrefix + long}
	line := ansi.Strip(m.noteLine(60))
	if len([]rune(line)) != 60 || !strings.HasPrefix(line, "Exported to …") || !strings.HasSuffix(line, "090000.json") {
		t.Errorf("path note = %q (%d runes)", line, len([]rune(line)))
	}
	m = statsModel{note: "Export failed: export file already exists: " + long, noteErr: true}
	line = ansi.Strip(m.noteLine(60))
	if len([]rune(line)) != 60 || !strings.HasPrefix(line, "Export failed: export file already exists") || !strings.HasSuffix(line, "…") {
		t.Errorf("error note = %q (%d runes)", line, len([]rune(line)))
	}
	if short := ansi.Strip((statsModel{note: "Exported to x.json"}).noteLine(60)); short != "Exported to x.json" {
		t.Errorf("short note changed: %q", short)
	}
}

var errDisk = errors.New("disk on fire")

// P4-03: an export directory that cannot be written shows the error on Stats; nothing is
// written, the screen stays, and the data is untouched.
func TestErrorExportDirNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	h, dir := exportHarness(t)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	before, err := h.store.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	h.press("e")
	v := ansi.Strip(h.view())
	if !strings.Contains(v, "Export failed:") || !strings.Contains(v, "permission denied") {
		t.Errorf("unwritable export dir: error not shown:\n%s", v)
	}
	if h.m.screen != screenStats || h.m.stats.exporting {
		t.Errorf("screen %d, exporting %v; want Stats, finished", h.m.screen, h.m.stats.exporting)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files written to an unwritable dir: %v", entries)
	}
	after, err := h.store.All(context.Background())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Errorf("tasks changed by a failed export: %v", err)
	}
}
