// Package alarm sends the time-up notification and plays the looping alarm sound
// (R-33, R-34, D-13) using the operating system's own tools. Every method returns at
// once: the tools run in the background, and their failures are logged, never returned.
package alarm

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Config holds the alarm's settings. Zero fields take the defaults New documents.
type Config struct {
	// GOOS picks the tools; default runtime.GOOS.
	GOOS string
	// Dir is where the sound is written (the data directory).
	Dir string
	// Sound is the alarm sound until SetSound changes it; default DefaultSound.
	Sound Sound
	// Log receives tool failures; default discards.
	Log *slog.Logger
	// Bell rings the terminal bell. It is the fallback when no player works (R-34);
	// default does nothing.
	Bell func()
	// BellEvery is the fallback bell's interval; default 2 s (R-34).
	BellEvery time.Duration
	// LookPath finds a tool; default exec.LookPath.
	LookPath func(name string) (string, error)
	// Exec runs a tool directly, never through a shell, until it exits or ctx ends;
	// default runCommand.
	Exec func(ctx context.Context, path string, args ...string) error
}

// notifyTimeout bounds one notification tool run.
const notifyTimeout = 10 * time.Second

// minPlay is the shortest player run counted as a success. A player that exits sooner
// (for example with no audio device) is treated as failed, so it is not restarted in a
// tight loop.
const minPlay = 200 * time.Millisecond

// Alarm implements the TUI's Alarm interface. At most one sound plays at a time: the
// looping alarm, or a one-off preview (D-16).
type Alarm struct {
	cfg     Config
	mu      sync.Mutex
	sound   Sound
	stop    chan struct{} // nil while silent
	done    chan struct{} // closed when the current sound has ended
	preview bool          // the current sound is a preview, not the alarm
	wg      sync.WaitGroup
}

// New returns an Alarm for cfg, filling in defaults.
func New(cfg Config) *Alarm {
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.Bell == nil {
		cfg.Bell = func() {}
	}
	if cfg.BellEvery <= 0 {
		cfg.BellEvery = 2 * time.Second
	}
	if cfg.LookPath == nil {
		cfg.LookPath = exec.LookPath
	}
	if cfg.Exec == nil {
		cfg.Exec = runCommand
	}
	if _, ok := ParseSound(string(cfg.Sound)); !ok {
		cfg.Sound = DefaultSound
	}
	return &Alarm{cfg: cfg, sound: cfg.Sound}
}

// runCommand runs path with args, with no stdin and no terminal output, and kills it when
// ctx ends.
func runCommand(ctx context.Context, path string, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.WaitDelay = time.Second
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) > 0 {
		return errors.Join(err, errors.New(string(out)))
	}
	return err
}

// notifyCommand is the notification tool and its arguments for goos (R-33, D-13). The
// task name is a separate argument, never part of a script or shell line.
func notifyCommand(goos, taskName string) (tool string, args []string, ok bool) {
	switch goos {
	case "darwin":
		return "osascript", []string{
			"-e", "on run argv",
			"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
			"-e", "end run",
			"--", "unitock", "Time's up: " + taskName,
		}, true
	case "linux":
		return "notify-send", []string{"--app-name=unitock", "--urgency=critical", "--", "Time's up", taskName}, true
	}
	return "", nil, false
}

// players are the sound players to try, in order, for goos (R-34).
func players(goos string) []string {
	switch goos {
	case "darwin":
		return []string{"afplay"}
	case "linux":
		return []string{"paplay", "aplay"}
	}
	return nil
}

// Notify sends one system notification that taskName's time is up.
func (a *Alarm) Notify(taskName string) {
	tool, args, ok := notifyCommand(a.cfg.GOOS, taskName)
	if !ok {
		a.cfg.Log.Warn("no notification tool for this OS", "goos", a.cfg.GOOS)
		return
	}
	path, err := a.cfg.LookPath(tool)
	if err != nil {
		a.cfg.Log.Warn("notification tool not found", "tool", tool, "err", err)
		return
	}
	a.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
		defer cancel()
		if err := a.cfg.Exec(ctx, path, args...); err != nil {
			a.cfg.Log.Warn("notification failed", "tool", tool, "err", err)
		}
	})
}

// SetSound sets the sound the next alarm plays (R-53). An unknown sound means
// DefaultSound. A loop already playing keeps its sound.
func (a *Alarm) SetSound(s Sound) {
	if _, ok := ParseSound(string(s)); !ok {
		s = DefaultSound
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sound = s
}

// Start begins the looping alarm sound, ending any preview first. It does nothing if
// the alarm is already playing.
func (a *Alarm) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stop != nil && !a.preview {
		return
	}
	a.play(a.sound, false)
}

// Preview plays s once (R-52), ending any earlier preview. It does nothing while the
// alarm is playing, so a preview never replaces it. Stop ends it.
func (a *Alarm) Preview(s Sound) {
	if _, ok := ParseSound(string(s)); !ok {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stop != nil && !a.preview {
		return
	}
	a.play(s, true)
}

// play ends the current sound, if any, and starts s: looping, or once for a preview.
// The new sound waits for the old one's player to exit, so two never write or play the
// sound file at once. a.mu must be held.
func (a *Alarm) play(s Sound, once bool) {
	prev := a.done
	if a.stop != nil {
		close(a.stop)
	}
	a.stop, a.done, a.preview = make(chan struct{}), make(chan struct{}), once
	stop, done := a.stop, a.done
	go func() {
		defer close(done)
		if prev != nil {
			<-prev
		}
		select {
		case <-stop: // stopped while waiting
			return
		default:
		}
		if once {
			a.once(s, stop)
		} else {
			a.loop(s, stop)
		}
	}()
}

// Stop ends the alarm sound without waiting for the player to exit. It does nothing if
// the sound is not playing.
func (a *Alarm) Stop() { a.stopLoop() }

// Close stops the sound and waits for the player and any notification to finish, so no
// tool outlives the app. Call it once before exit.
func (a *Alarm) Close() {
	if done := a.stopLoop(); done != nil {
		<-done
	}
	a.wg.Wait()
}

func (a *Alarm) stopLoop() (done chan struct{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stop == nil {
		return nil
	}
	close(a.stop)
	done = a.done
	a.stop, a.done, a.preview = nil, nil, false
	return done
}

// prepare finds a player and writes s for it, returning the player's path, the sound
// file, and a context cancelled when stop closes. It returns ok false, having logged
// why, when the bell must stand in.
func (a *Alarm) prepare(s Sound, stop chan struct{}) (player, wav string, ctx context.Context, cancel context.CancelFunc, ok bool) {
	player = a.player()
	if player == "" {
		return "", "", nil, nil, false
	}
	wav, err := writeSound(a.cfg.Dir, s)
	if err != nil {
		a.cfg.Log.Warn("writing alarm sound failed; using the terminal bell", "err", err)
		return "", "", nil, nil, false
	}
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	return player, wav, ctx, cancel, true
}

// once plays s a single time for a preview (R-52). With no player, or if it fails, it
// rings the bell once.
func (a *Alarm) once(s Sound, stop chan struct{}) {
	player, wav, ctx, cancel, ok := a.prepare(s, stop)
	if !ok {
		a.cfg.Bell()
		return
	}
	defer cancel()
	if err := a.cfg.Exec(ctx, player, wav); err != nil && ctx.Err() == nil {
		a.cfg.Log.Warn("sound preview failed", "player", player, "sound", s, "err", err)
		a.cfg.Bell()
	}
}

// loop plays s with the first available player until stop closes. With no player, or
// once the player fails, it rings the bell instead.
func (a *Alarm) loop(s Sound, stop chan struct{}) {
	player, wav, ctx, cancel, ok := a.prepare(s, stop)
	if !ok {
		a.bell(stop)
		return
	}
	defer cancel()
	for {
		began := time.Now()
		err := a.cfg.Exec(ctx, player, wav)
		if ctx.Err() != nil {
			return
		}
		if err == nil && time.Since(began) < minPlay {
			err = errors.New("player exited at once")
		}
		if err != nil {
			a.cfg.Log.Warn("alarm player failed; using the terminal bell", "player", player, "err", err)
			a.bell(stop)
			return
		}
	}
}

// player is the path of the first sound player found, or "" with none.
func (a *Alarm) player() string {
	for _, p := range players(a.cfg.GOOS) {
		if path, err := a.cfg.LookPath(p); err == nil {
			return path
		}
	}
	a.cfg.Log.Warn("no alarm player found; using the terminal bell", "goos", a.cfg.GOOS, "tried", players(a.cfg.GOOS))
	return ""
}

// bell rings at once, then every BellEvery until stop closes (R-34).
func (a *Alarm) bell(stop chan struct{}) {
	t := time.NewTicker(a.cfg.BellEvery)
	defer t.Stop()
	for {
		a.cfg.Bell()
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}
