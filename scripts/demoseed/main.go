// Command demoseed fills a scratch data directory with about six months of realistic
// task history for the README demos (`make demo`). It writes through the store's own
// API, so the data is exactly what the app would have stored. It refuses a directory
// that already has a database, so it can never touch real data.
//
//	go run ./scripts/demoseed --data-dir DIR [--running SECONDS]
//
// --running also leaves one active task that runs out SECONDS after the seed, for the
// time-up alert demo (tasks are at least a minute long, too long to wait for in a GIF).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/azntaiji/unitock/internal/store"
)

func main() {
	dir := flag.String("data-dir", "", "scratch data directory to create and fill (required)")
	running := flag.Int("running", 0, "leave an active task that runs out this many seconds from now")
	flag.Parse()
	if err := run(context.Background(), *dir, time.Now(), time.Duration(*running)*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, "demoseed:", err)
		os.Exit(1)
	}
}

// work is what a demo task can be: its project (empty for none), name, and typical
// estimate in minutes.
var work = []struct {
	project, name string
	minutes       int
}{
	{"unitock", "Review pull requests", 30},
	{"unitock", "Write release notes", 45},
	{"unitock", "Fix flaky test", 60},
	{"unitock", "Refactor stats screen", 90},
	{"Writing", "Draft blog post", 60},
	{"Writing", "Edit chapter 3", 45},
	{"Writing", "Outline talk", 30},
	{"Admin", "Inbox zero", 25},
	{"Admin", "Expense report", 20},
	{"", "Read papers", 40},
	{"", "Plan the week", 15},
}

func run(ctx context.Context, dir string, now time.Time, running time.Duration) error {
	if dir == "" {
		return errors.New("--data-dir is required")
	}
	if _, err := os.Stat(filepath.Join(dir, store.DBFile)); err == nil {
		return fmt.Errorf("%s already has a database; use an empty scratch directory", dir)
	}
	st, err := store.Open(ctx, dir)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	r := rand.New(rand.NewPCG(1, 2)) // the same history every run
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	n := 0
	// Six months, so the 3-month Stats period has a previous period to compare with;
	// today counts too, with only the tasks already finished by now.
	for day := 182; day >= 0; day-- {
		date := today.AddDate(0, 0, -day)
		if wd := date.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		at := date.Add(9*time.Hour + time.Duration(r.IntN(60))*time.Minute)
		for range 2 + r.IntN(3) {
			w := work[r.IntN(len(work))]
			est := time.Duration(w.minutes) * time.Minute
			// Most tasks run over a little; some finish early.
			actual := est * time.Duration(80+r.IntN(60)) / 100
			if at.Add(actual + 5*time.Minute).After(now) {
				break
			}
			t, err := st.Create(ctx, w.project, w.name, est, at)
			if err != nil {
				return err
			}
			end := at.Add(actual)
			if actual > est && r.IntN(2) == 0 { // extended at time up
				if _, err := st.AddExtension(ctx, t.ID, at.Add(est), actual-est+5*time.Minute); err != nil {
					return err
				}
			}
			if r.IntN(5) == 0 { // a short break
				p := at.Add(actual / 2)
				if _, err := st.Pause(ctx, t.ID, p); err != nil {
					return err
				}
				if _, err := st.Resume(ctx, t.ID, p.Add(5*time.Minute)); err != nil {
					return err
				}
				end = end.Add(5 * time.Minute)
			}
			if r.IntN(15) == 0 {
				err = st.Cancel(ctx, t.ID, end)
			} else {
				err = st.Complete(ctx, t.ID, end)
			}
			if err != nil {
				return err
			}
			n++
			at = end.Add(time.Duration(10+r.IntN(50)) * time.Minute)
		}
	}
	if running > 0 {
		est := 5 * time.Minute
		if _, err := st.Create(ctx, "", "Stretch break", est, now.Add(running-est)); err != nil {
			return err
		}
		n++
	}
	fmt.Printf("seeded %d tasks in %s\n", n, dir)
	return nil
}
