// Package export writes every task to a JSON file in schema version 1 (R-50, D-14).
// The format is the public way to read unitock's data: fields are only ever added in
// 1.x; renames or removals bump SchemaVersion.
package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/azntaiji/unitock/internal/task"
)

// SchemaVersion is the export format version (D-14).
const SchemaVersion = 1

// ErrExists is returned when the export file already exists; it is never overwritten.
var ErrExists = errors.New("export file already exists")

// Document is the top-level JSON object.
type Document struct {
	SchemaVersion int    `json:"schema_version"`
	AppVersion    string `json:"app_version"`
	ExportedAt    string `json:"exported_at"`
	Tasks         []Task `json:"tasks"`
}

// Task is one exported task. Times are RFC 3339 with offset; durations whole seconds.
type Task struct {
	ID                      int64       `json:"id"`
	Name                    string      `json:"name"`
	Project                 *string     `json:"project"` // null without a project
	Status                  string      `json:"status"`
	StartedAt               string      `json:"started_at"`
	EndedAt                 *string     `json:"ended_at"` // null while active
	OriginalEstimateSeconds int64       `json:"original_estimate_seconds"`
	EstimateSeconds         int64       `json:"estimate_seconds"`
	ActualSeconds           int64       `json:"actual_seconds"` // elapsed at export time
	PausedSeconds           int64       `json:"paused_seconds"`
	Extensions              []Extension `json:"extensions"`
	Pauses                  []Pause     `json:"pauses"`
}

// Extension is time added to a task's estimate.
type Extension struct {
	At      string `json:"at"`
	Seconds int64  `json:"seconds"`
}

// Pause is one pause; EndedAt is null for a pause still open.
type Pause struct {
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at"`
}

// Build converts tasks to a Document as of now. Times are shown in now's location.
func Build(tasks []task.Task, appVersion string, now time.Time) Document {
	loc := now.Location()
	stamp := func(t time.Time) string { return t.In(loc).Format(time.RFC3339) }
	doc := Document{SchemaVersion: SchemaVersion, AppVersion: appVersion, ExportedAt: stamp(now), Tasks: []Task{}}
	for _, t := range tasks {
		end := now
		out := Task{
			ID:                      t.ID,
			Name:                    t.Name,
			Status:                  string(t.Status),
			StartedAt:               stamp(t.StartedAt),
			OriginalEstimateSeconds: seconds(t.OriginalEstimate),
			EstimateSeconds:         seconds(t.Estimate()),
			Extensions:              []Extension{},
			Pauses:                  []Pause{},
		}
		if t.Project != "" {
			p := t.Project
			out.Project = &p
		}
		if !t.EndedAt.IsZero() {
			e := stamp(t.EndedAt)
			out.EndedAt, end = &e, t.EndedAt
		}
		elapsed := t.Elapsed(now)
		out.ActualSeconds = seconds(elapsed)
		out.PausedSeconds = seconds(end.Sub(t.StartedAt) - elapsed)
		for _, e := range t.Extensions {
			out.Extensions = append(out.Extensions, Extension{At: stamp(e.At), Seconds: seconds(e.Added)})
		}
		for _, p := range t.Pauses {
			ep := Pause{StartedAt: stamp(p.StartedAt)}
			if !p.Open() {
				e := stamp(p.EndedAt)
				ep.EndedAt = &e
			}
			out.Pauses = append(out.Pauses, ep)
		}
		doc.Tasks = append(doc.Tasks, out)
	}
	return doc
}

func seconds(d time.Duration) int64 { return int64(d / time.Second) }

// FileName is the export file name for now: unitock-export-YYYYMMDD-HHMMSS.json in
// now's (local) time (R-50).
func FileName(now time.Time) string {
	return "unitock-export-" + now.Format("20060102-150405") + ".json"
}

// Write writes the export of tasks as of now into dir and returns the file's path. It
// never overwrites: if the file exists it returns ErrExists and leaves it alone (R-50).
func Write(dir string, tasks []task.Task, appVersion string, now time.Time) (string, error) {
	body, err := json.MarshalIndent(Build(tasks, appVersion, now), "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding export: %w", err)
	}
	path := filepath.Join(dir, FileName(now))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("%w: %s", ErrExists, path)
	}
	if err != nil {
		return "", fmt.Errorf("creating export: %w", err)
	}
	if _, err := f.Write(append(body, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("writing export: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("writing export: %w", err)
	}
	return path, nil
}
