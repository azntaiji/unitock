package export

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/azntaiji/unitock/internal/task"
)

var la = func() *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}
	return loc
}()

func at(h, m, s int) time.Time { return time.Date(2026, 11, 17, h, m, s, 0, la) }

// fixture holds the docs/SPEC.md example task, an active task with an open pause and
// no project, and a cancelled task, exported at 14:03 local.
func fixture() ([]task.Task, time.Time) {
	return []task.Task{
		{
			ID: 1, Name: "Write spec", Project: "unitock", Status: task.Completed,
			StartedAt: at(13, 0, 0), EndedAt: at(13, 52, 10), OriginalEstimate: 2700 * time.Second,
			Extensions: []task.Extension{{At: at(13, 45, 0), Added: 900 * time.Second}},
			Pauses:     []task.Pause{{StartedAt: at(13, 20, 0), EndedAt: at(13, 23, 0)}},
		},
		{
			ID: 2, Name: "Abandoned", Status: task.Cancelled,
			StartedAt: at(13, 53, 0).UTC(), EndedAt: at(13, 55, 30).UTC(), OriginalEstimate: 10 * time.Minute,
		},
		{
			ID: 3, Name: "Review \"draft\" ✓", Status: task.Active,
			StartedAt: at(13, 56, 0), OriginalEstimate: 25 * time.Minute,
			Pauses: []task.Pause{{StartedAt: at(14, 1, 0)}},
		},
	}, at(14, 3, 0)
}

// P3-05: the export of the fixture matches the reviewed schema v1 golden file, whose
// first task is the docs/SPEC.md example (D-14).
func TestExportGolden(t *testing.T) {
	tasks, now := fixture()
	dir := t.TempDir()
	path, err := Write(dir, tasks, "1.0.0", now)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "unitock-export-20261117-140300.json"); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "export_v1.golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run `make golden` to create it)", err)
	}
	if string(got) != string(want) {
		t.Errorf("export differs from %s\n--- got\n%s\n--- want\n%s", golden, got, want)
	}
}

// The first task's values are the docs/SPEC.md example's, independent of the golden.
func TestExportSpecExampleValues(t *testing.T) {
	tasks, now := fixture()
	doc := Build(tasks, "1.0.0", now)
	if doc.SchemaVersion != 1 || doc.ExportedAt != "2026-11-17T14:03:00-08:00" || len(doc.Tasks) != 3 {
		t.Fatalf("header = %d %s %d tasks", doc.SchemaVersion, doc.ExportedAt, len(doc.Tasks))
	}
	got := doc.Tasks[0]
	if *got.Project != "unitock" || *got.EndedAt != "2026-11-17T13:52:10-08:00" ||
		got.OriginalEstimateSeconds != 2700 || got.EstimateSeconds != 3600 ||
		got.ActualSeconds != 2950 || got.PausedSeconds != 180 {
		t.Errorf("spec example task = %+v", got)
	}
	cancelled, active := doc.Tasks[1], doc.Tasks[2]
	if cancelled.StartedAt != "2026-11-17T13:53:00-08:00" {
		t.Errorf("UTC-stored time not shown local: %s", cancelled.StartedAt)
	}
	// Active: 7 minutes since start, 2 of them in the open pause.
	if active.Project != nil || active.EndedAt != nil || active.ActualSeconds != 300 ||
		active.PausedSeconds != 120 || active.Pauses[0].EndedAt != nil {
		t.Errorf("active task = %+v", active)
	}
	// Empty lists are [] not null, so readers need not special-case them.
	b, _ := json.Marshal(cancelled)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["extensions"] == nil || m["pauses"] == nil || m["project"] != nil {
		t.Errorf("cancelled task JSON = %s", b)
	}
}

// R-50: an existing file is never overwritten.
func TestExportNeverOverwrites(t *testing.T) {
	tasks, now := fixture()
	dir := t.TempDir()
	path := filepath.Join(dir, FileName(now))
	if err := os.WriteFile(path, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(dir, tasks, "1.0.0", now); !errors.Is(err, ErrExists) {
		t.Errorf("Write over an existing file: err = %v, want ErrExists", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "keep me" {
		t.Errorf("existing file changed to %q", b)
	}
}

func TestExportMissingDirErrors(t *testing.T) {
	tasks, now := fixture()
	if _, err := Write(filepath.Join(t.TempDir(), "nope"), tasks, "1.0.0", now); err == nil {
		t.Error("Write into a missing directory succeeded")
	}
}

func TestExportEmpty(t *testing.T) {
	b, err := json.Marshal(Build(nil, "dev", at(9, 0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":1,"app_version":"dev","exported_at":"2026-11-17T09:00:00-08:00","tasks":[]}`
	if string(b) != want {
		t.Errorf("empty export = %s, want %s", b, want)
	}
}
