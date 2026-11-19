package tui

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/azntaiji/unitock/internal/store"
	"github.com/azntaiji/unitock/internal/task"
)

// probe wraps the model under a real Bubble Tea program and records its rendered view
// after every update. The terminal output stream only carries changed cells, so text
// such as a countdown is checked here rather than in the raw bytes.
type probe struct {
	inner tea.Model
	mu    *sync.Mutex
	view  *string
}

func newProbe(m tea.Model) probe {
	p := probe{inner: m, mu: &sync.Mutex{}, view: new(string)}
	p.record()
	return p
}

func (p probe) record() {
	v := p.inner.View().Content
	p.mu.Lock()
	*p.view = v
	p.mu.Unlock()
}

func (p probe) Init() tea.Cmd { return p.inner.Init() }

func (p probe) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	p.inner, cmd = p.inner.Update(msg)
	p.record()
	return p, cmd
}

func (p probe) View() tea.View { return p.inner.View() }

func (p probe) current() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return *p.view
}

// waitView waits until the latest rendered view contains want.
func (p probe) waitView(t *testing.T, want string, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if strings.Contains(p.current(), want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("view never contained %q; last view:\n%s", want, p.current())
}

type app struct {
	tm    *teatest.TestModel
	probe probe
	store *store.Store
}

// launch opens the store in dir and runs the UI under teatest, as a process launch would.
func launch(t *testing.T, dir string, clk task.Clock, rnd func(int) int) app {
	t.Helper()
	st, err := store.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	m, err := New(Deps{Store: st, Clock: clk, Rand: rnd})
	if err != nil {
		t.Fatal(err)
	}
	p := newProbe(m)
	tm := teatest.NewTestModel(t, p, teatest.WithInitialTermSize(80, 24))
	return app{tm: tm, probe: p, store: st}
}

func (a app) press(code rune) { a.tm.Send(tea.KeyPressMsg{Code: code}) }

func (a app) quit(t *testing.T) {
	t.Helper()
	a.tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}) // quits from any screen (R-06)
	a.tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestFlowStartCountdownComplete(t *testing.T) {
	clk := task.NewFakeClock(t0)
	a := launch(t, t.TempDir(), clk, func(int) int { return 3 })

	teatest.WaitFor(t, a.tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Task name")) && bytes.Contains(b, []byte("Start task"))
	}, teatest.WithDuration(3*time.Second))

	a.tm.Type("unitock")
	a.press(tea.KeyTab) // no stored projects to complete: tab moves on
	a.tm.Type("Write report")
	a.press(tea.KeyTab)
	a.tm.Type("25m")
	a.press(tea.KeyEnter)
	a.probe.waitView(t, "25:00", 3*time.Second)
	a.probe.waitView(t, "Write report", time.Second)

	// No input: the program's own one-second tick must redraw from the clock (R-19).
	clk.Advance(10 * time.Second)
	a.probe.waitView(t, "Elapsed 00:10", 3*time.Second) // 24:50 left
	clk.Advance(5 * time.Minute)
	a.probe.waitView(t, "Elapsed 05:10", 3*time.Second) // 19:50 left

	a.tm.Type("e")
	a.probe.waitView(t, messages[3], 3*time.Second)
	a.tm.Type("x")
	a.probe.waitView(t, "Task name", 3*time.Second)
	a.quit(t)

	got, err := a.store.Get(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Write report" || got.Project != "unitock" || got.Status != task.Completed ||
		!got.EndedAt.Equal(t0.Add(5*time.Minute+10*time.Second)) || got.OriginalEstimate != 25*time.Minute {
		t.Errorf("stored task = %+v", got)
	}
}

func TestResumeTeatestQuitAndRelaunch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	clk := task.NewFakeClock(t0)

	a := launch(t, dir, clk, nil)
	a.press(tea.KeyTab) // skip the project
	a.tm.Type("Deep work")
	a.press(tea.KeyTab)
	a.tm.Type("2m")
	a.press(tea.KeyEnter)
	a.probe.waitView(t, "02:00", 3*time.Second)
	clk.Advance(5 * time.Second)
	a.probe.waitView(t, "Elapsed 00:05", 3*time.Second) // 01:55 left
	a.quit(t)
	_ = a.store.Close()

	// Closed for 20 seconds (R-04, R-05): the time still counts.
	clk.Advance(20 * time.Second)
	b := launch(t, dir, clk, nil)
	b.probe.waitView(t, "Deep work", 3*time.Second)
	b.probe.waitView(t, "Elapsed 00:25", 3*time.Second) // 01:35 left
	b.quit(t)

	got, err := b.store.Active(context.Background())
	if err != nil || got.Name != "Deep work" {
		t.Errorf("after relaunch and quit, Active = %+v, %v", got, err)
	}
}
