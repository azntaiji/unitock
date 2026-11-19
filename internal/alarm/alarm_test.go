package alarm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// run is one recorded tool run.
type run struct {
	path string
	args []string
}

// fakeOS records tool runs. Tools in have are found at /bin/<name>. A player run blocks
// until killed unless failPlayer is set; notifications return at once.
type fakeOS struct {
	have       []string
	failPlayer bool
	mu         sync.Mutex
	runs       []run
	started    chan run
	killed     chan string
	bells      atomic.Int32
	log        bytes.Buffer
	logMu      sync.Mutex
}

func newFakeOS(have ...string) *fakeOS {
	return &fakeOS{have: have, started: make(chan run, 10), killed: make(chan string, 10)}
}

func (f *fakeOS) Write(p []byte) (int, error) {
	f.logMu.Lock()
	defer f.logMu.Unlock()
	return f.log.Write(p)
}

func (f *fakeOS) logText() string {
	f.logMu.Lock()
	defer f.logMu.Unlock()
	return f.log.String()
}

func (f *fakeOS) config(t *testing.T, goos string) Config {
	return Config{
		GOOS:      goos,
		Dir:       t.TempDir(),
		Log:       slog.New(slog.NewTextHandler(f, nil)),
		Bell:      func() { f.bells.Add(1) },
		BellEvery: 10 * time.Millisecond,
		LookPath: func(name string) (string, error) {
			if slices.Contains(f.have, name) {
				return "/bin/" + name, nil
			}
			return "", errors.New("not found")
		},
		Exec: func(ctx context.Context, path string, args ...string) error {
			r := run{path, args}
			f.mu.Lock()
			f.runs = append(f.runs, r)
			f.mu.Unlock()
			f.started <- r
			if isPlayer(path) {
				if f.failPlayer {
					return errors.New("no audio device")
				}
				<-ctx.Done()
				f.killed <- path
				return ctx.Err()
			}
			return nil
		},
	}
}

func isPlayer(path string) bool {
	return slices.Contains([]string{"/bin/afplay", "/bin/paplay", "/bin/aplay"}, path)
}

func (f *fakeOS) waitRun(t *testing.T) run {
	t.Helper()
	select {
	case r := <-f.started:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("no tool was run")
		return run{}
	}
}

// A task name that would break out of a script or shell line if it were interpolated.
const hostile = `x" & (do shell script "touch /tmp/pwned") & "'; rm -rf ~ #`

func TestNotifyMacOSPassesNameAsArgument(t *testing.T) {
	f := newFakeOS("osascript")
	a := New(f.config(t, "darwin"))
	a.Notify(hostile)
	r := f.waitRun(t)
	a.Close()
	if r.path != "/bin/osascript" {
		t.Fatalf("ran %q, want osascript", r.path)
	}
	if got := r.args[len(r.args)-1]; got != "Time's up: "+hostile {
		t.Errorf("last argument %q, want the task name as its own argument", got)
	}
	for _, arg := range r.args[:len(r.args)-1] {
		if strings.Contains(arg, "rm -rf") || strings.Contains(arg, "do shell script") {
			t.Errorf("task name interpolated into %q", arg)
		}
	}
	if !slices.Contains(r.args, "on run argv") || !slices.Contains(r.args, "--") {
		t.Errorf("script must read argv and end options before the name: %q", r.args)
	}
}

func TestNotifyLinuxPassesNameAsArgument(t *testing.T) {
	f := newFakeOS("notify-send")
	a := New(f.config(t, "linux"))
	a.Notify("-e " + hostile)
	r := f.waitRun(t)
	a.Close()
	if r.path != "/bin/notify-send" {
		t.Fatalf("ran %q, want notify-send", r.path)
	}
	n := len(r.args)
	if r.args[n-1] != "-e "+hostile || r.args[n-3] != "--" {
		t.Errorf("args %q: want `--`, a summary, then the task name as its own argument", r.args)
	}
}

func TestNotifyMissingToolIsLogged(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "plan9"} {
		f := newFakeOS()
		a := New(f.config(t, goos))
		a.Notify("write")
		a.Close()
		if len(f.runs) != 0 || !strings.Contains(f.logText(), "notification") {
			t.Errorf("%s: runs %v, log %q; want nothing run and a log line", goos, f.runs, f.logText())
		}
	}
}

func TestNotifyFailureIsLogged(t *testing.T) {
	f := newFakeOS("osascript")
	cfg := f.config(t, "darwin")
	cfg.Exec = func(context.Context, string, ...string) error { return errors.New("denied") }
	a := New(cfg)
	a.Notify("write")
	a.Close()
	if !strings.Contains(f.logText(), "notification failed") || !strings.Contains(f.logText(), "denied") {
		t.Errorf("log %q", f.logText())
	}
}

func TestPlayerPerOS(t *testing.T) {
	for _, tt := range []struct {
		goos string
		have []string
		want string
	}{
		{"darwin", []string{"afplay", "paplay"}, "/bin/afplay"},
		{"linux", []string{"aplay", "paplay"}, "/bin/paplay"},
		{"linux", []string{"aplay"}, "/bin/aplay"},
	} {
		f := newFakeOS(tt.have...)
		cfg := f.config(t, tt.goos)
		a := New(cfg)
		a.Start()
		r := f.waitRun(t)
		if r.path != tt.want {
			t.Errorf("%s with %v: played with %q, want %q", tt.goos, tt.have, r.path, tt.want)
		}
		wav := filepath.Join(cfg.Dir, ChimeFile)
		if !slices.Equal(r.args, []string{wav}) {
			t.Errorf("player args %q, want the chime %q", r.args, wav)
		}
		if b, err := os.ReadFile(wav); err != nil || !bytes.HasPrefix(b, []byte("RIFF")) {
			t.Errorf("chime not written: %v", err)
		}
		a.Close()
		if got := <-f.killed; got != tt.want {
			t.Errorf("killed %q", got)
		}
		if f.bells.Load() != 0 {
			t.Errorf("bell rang %d times while a player worked", f.bells.Load())
		}
	}
}

func TestStartIsIdempotentAndStopKills(t *testing.T) {
	f := newFakeOS("afplay")
	a := New(f.config(t, "darwin"))
	a.Start()
	a.Start()
	f.waitRun(t)
	a.Stop()
	select {
	case <-f.killed:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not kill the player")
	}
	a.Stop() // already stopped: no effect
	a.Start()
	f.waitRun(t)
	a.Close()
	if n := len(f.runs); n != 2 {
		t.Errorf("%d player runs, want 2 (one per Start while silent)", n)
	}
}

func TestPlayerLoopsUntilStopped(t *testing.T) {
	f := newFakeOS("afplay")
	cfg := f.config(t, "darwin")
	var plays atomic.Int32
	cfg.Exec = func(ctx context.Context, path string, args ...string) error {
		plays.Add(1)
		select { // one play of the chime
		case <-time.After(minPlay + 10*time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	a := New(cfg)
	a.Start()
	deadline := time.Now().Add(3 * time.Second)
	for plays.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	a.Close()
	n := plays.Load()
	if n < 3 {
		t.Fatalf("chime played %d times, want it to loop", n)
	}
	time.Sleep(minPlay + 50*time.Millisecond)
	if plays.Load() != n {
		t.Error("chime kept playing after Close")
	}
}

func TestBellWhenNoPlayer(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "plan9"} {
		f := newFakeOS("osascript", "notify-send")
		a := New(f.config(t, goos))
		a.Start()
		waitBells(t, f, 3)
		a.Close()
		n := f.bells.Load()
		time.Sleep(30 * time.Millisecond)
		if f.bells.Load() != n {
			t.Errorf("%s: bell kept ringing after Close", goos)
		}
		if len(f.runs) != 0 || !strings.Contains(f.logText(), "terminal bell") {
			t.Errorf("%s: runs %v, log %q", goos, f.runs, f.logText())
		}
	}
}

func TestBellWhenPlayerFails(t *testing.T) {
	f := newFakeOS("paplay")
	f.failPlayer = true
	a := New(f.config(t, "linux"))
	a.Start()
	waitBells(t, f, 2)
	a.Close()
	if !strings.Contains(f.logText(), "no audio device") {
		t.Errorf("log %q", f.logText())
	}
}

func TestBellWhenPlayerExitsAtOnce(t *testing.T) {
	f := newFakeOS("aplay")
	cfg := f.config(t, "linux")
	cfg.Exec = func(context.Context, string, ...string) error { return nil }
	a := New(cfg)
	a.Start()
	waitBells(t, f, 2)
	a.Close()
	if !strings.Contains(f.logText(), "exited at once") {
		t.Errorf("log %q", f.logText())
	}
}

func TestBellWhenSoundUnwritable(t *testing.T) {
	f := newFakeOS("afplay")
	cfg := f.config(t, "darwin")
	cfg.Dir = filepath.Join(cfg.Dir, "missing")
	a := New(cfg)
	a.Start()
	waitBells(t, f, 1)
	a.Close()
	if len(f.runs) != 0 || !strings.Contains(f.logText(), "alarm sound") {
		t.Errorf("runs %v, log %q", f.runs, f.logText())
	}
}

func waitBells(t *testing.T, f *fakeOS, n int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for f.bells.Load() < n {
		if time.Now().After(deadline) {
			t.Fatalf("bell rang %d times, want at least %d", f.bells.Load(), n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// P4-03: with no notification tool and no audio player at all, nothing fails: each
// problem is logged naming the missing tool, the alarm falls back to the terminal bell,
// a preview rings it once, and Close returns.
func TestErrorMissingAudioAndNotifyTools(t *testing.T) {
	for _, tc := range []struct{ goos, notifier, player string }{
		{"darwin", "osascript", "afplay"},
		{"linux", "notify-send", "paplay"},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			f := newFakeOS() // no tools at all
			a := New(f.config(t, tc.goos))
			a.Notify("write report")
			a.Start()
			waitBells(t, f, 2)
			a.Stop()
			rung := f.bells.Load()
			a.Preview(Bell)
			waitBells(t, f, rung+1)
			a.Close()
			if len(f.runs) != 0 {
				t.Errorf("ran %v with no tools installed", f.runs)
			}
			log := f.logText()
			for _, want := range []string{"notification tool not found", tc.notifier, "no alarm player found", tc.player, "terminal bell"} {
				if !strings.Contains(log, want) {
					t.Errorf("log lacks %q:\n%s", want, log)
				}
			}
		})
	}
}
