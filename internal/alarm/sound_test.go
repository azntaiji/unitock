package alarm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSoundsArePCMWAV(t *testing.T) {
	for _, s := range Sounds {
		t.Run(string(s), func(t *testing.T) {
			b := s.WAV()
			if string(b[0:4]) != "RIFF" || string(b[8:16]) != "WAVEfmt " || string(b[36:40]) != "data" {
				t.Fatalf("bad header % x", b[:44])
			}
			if got := binary.LittleEndian.Uint32(b[4:8]); int(got) != len(b)-8 {
				t.Errorf("RIFF size %d, file %d", got, len(b))
			}
			if got := binary.LittleEndian.Uint32(b[40:44]); int(got) != len(b)-44 {
				t.Errorf("data size %d, want %d", got, len(b)-44)
			}
			if rate := binary.LittleEndian.Uint32(b[24:28]); rate != sampleRate {
				t.Errorf("rate %d", rate)
			}
			if secs := float64(len(b)-44) / 2 / sampleRate; secs < 1 || secs > 3 {
				t.Errorf("%.2f s long, want 1 to 3 s so the loop repeats promptly", secs)
			}
			var peak, clipped int
			for i := 44; i+1 < len(b); i += 2 {
				v := int(int16(binary.LittleEndian.Uint16(b[i:])))
				peak = max(peak, v, -v)
				if v == math.MaxInt16 || v == -math.MaxInt16 {
					clipped++
				}
			}
			if peak < 5000 {
				t.Errorf("peak %d: near silent", peak)
			}
			if clipped > 0 {
				t.Errorf("%d clipped samples", clipped)
			}
		})
	}
}

// Chime is the D-13 sound and the default, byte for byte as before D-16.
func TestSoundChimeUnchanged(t *testing.T) {
	const want = "0d621ec6b795237d9d79489230f1e297b9d9654e95b002045c69471147a9c20a"
	if sum := sha256.Sum256(Chime.WAV()); hex.EncodeToString(sum[:]) != want {
		t.Errorf("Chime changed: sha256 %x", sum)
	}
	if DefaultSound != Chime {
		t.Errorf("DefaultSound = %q", DefaultSound)
	}
}

func TestSoundsAreDistinct(t *testing.T) {
	seen := map[string]Sound{}
	for _, s := range Sounds {
		k := string(s.WAV())
		if other, ok := seen[k]; ok {
			t.Errorf("%s and %s are the same sound", s, other)
		}
		seen[k] = s
	}
	if len(Sounds) != 5 {
		t.Errorf("%d sounds, want 5 (D-16)", len(Sounds))
	}
}

func TestParseSound(t *testing.T) {
	for _, s := range Sounds {
		if got, ok := ParseSound(string(s)); !ok || got != s {
			t.Errorf("ParseSound(%q) = %q, %v", s, got, ok)
		}
		if s.Label() == "" || s.Label() == string(s) {
			t.Errorf("%s has no display label", s)
		}
	}
	for _, bad := range []string{"", "Chime", "siren", " bell"} {
		if _, ok := ParseSound(bad); ok {
			t.Errorf("ParseSound(%q) accepted", bad)
		}
	}
	if !bytes.Equal(Sound("siren").WAV(), Chime.WAV()) {
		t.Error("an unknown sound does not play Chime")
	}
}

func TestWriteSoundOnce(t *testing.T) {
	dir := t.TempDir()
	p, err := writeSound(dir, Bell)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSound(dir, Bell); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(p); !st.ModTime().Equal(old) {
		t.Error("an identical sound was rewritten")
	}
	if _, err := writeSound(dir, Marimba); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, Marimba.WAV()) {
		t.Error("a different sound was not written over the old one")
	}
	if err := os.WriteFile(p, []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSound(dir, Marimba); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, Marimba.WAV()) {
		t.Error("a damaged sound was not replaced")
	}
}

// soundOS is a fakeOS whose player records which sound it was given, read from the
// file at the moment it starts. A play lasts playFor, or until killed when zero.
type soundOS struct {
	*fakeOS
	playFor time.Duration
	mu      sync.Mutex
	played  []Sound
	plays   chan Sound
	kills   atomic.Int32
}

func newSoundOS(t *testing.T, playFor time.Duration) (*soundOS, Config) {
	f := &soundOS{fakeOS: newFakeOS("afplay"), playFor: playFor, plays: make(chan Sound, 20)}
	cfg := f.config(t, "darwin")
	cfg.Exec = func(ctx context.Context, path string, args ...string) error {
		b, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		var s Sound
		for _, c := range Sounds {
			if bytes.Equal(b, c.WAV()) {
				s = c
			}
		}
		f.mu.Lock()
		f.played = append(f.played, s)
		f.mu.Unlock()
		f.plays <- s
		if f.playFor > 0 {
			select {
			case <-time.After(f.playFor):
				return nil
			case <-ctx.Done():
			}
		} else {
			<-ctx.Done()
		}
		f.kills.Add(1)
		return ctx.Err()
	}
	return f, cfg
}

func (f *soundOS) next(t *testing.T) Sound {
	t.Helper()
	select {
	case s := <-f.plays:
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was played")
		return ""
	}
}

func (f *soundOS) quiet(t *testing.T, d time.Duration) {
	t.Helper()
	select {
	case s := <-f.plays:
		t.Fatalf("%s played, want nothing", s)
	case <-time.After(d):
	}
}

func TestSoundSetChangesWhatTheAlarmPlays(t *testing.T) {
	f, cfg := newSoundOS(t, 0)
	a := New(cfg)
	a.Start()
	if s := f.next(t); s != Chime {
		t.Errorf("default alarm played %s, want chime", s)
	}
	a.SetSound(Rising) // a loop already playing keeps its sound
	a.Stop()
	a.Start()
	if s := f.next(t); s != Rising {
		t.Errorf("after SetSound(rising) the alarm played %s", s)
	}
	a.SetSound("siren") // unknown: back to the default
	a.Stop()
	a.Start()
	if s := f.next(t); s != Chime {
		t.Errorf("after an unknown sound the alarm played %s, want chime", s)
	}
	a.Close()

	f2, cfg2 := newSoundOS(t, 0)
	cfg2.Sound = Beep
	b := New(cfg2)
	b.Start()
	if s := f2.next(t); s != Beep {
		t.Errorf("Config.Sound beep: played %s", s)
	}
	b.Close()
}

func TestSoundPreviewPlaysOnce(t *testing.T) {
	f, cfg := newSoundOS(t, minPlay+20*time.Millisecond)
	a := New(cfg)
	a.Preview(Marimba)
	if s := f.next(t); s != Marimba {
		t.Fatalf("preview played %s", s)
	}
	f.quiet(t, 3*minPlay) // it does not loop
	a.Close()
	if f.bells.Load() != 0 {
		t.Errorf("bell rang %d times on a working preview", f.bells.Load())
	}
}

func TestSoundPreviewReplacesPreviewAndStopEndsIt(t *testing.T) {
	f, cfg := newSoundOS(t, 0)
	a := New(cfg)
	a.Preview(Bell)
	if s := f.next(t); s != Bell {
		t.Fatalf("first preview played %s", s)
	}
	a.Preview(Beep) // ends Bell first
	if s := f.next(t); s != Beep {
		t.Fatalf("second preview played %s", s)
	}
	if f.kills.Load() != 1 {
		t.Errorf("%d players killed, want the first preview's", f.kills.Load())
	}
	a.Stop()
	deadline := time.Now().Add(2 * time.Second)
	for f.kills.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if f.kills.Load() != 2 {
		t.Error("Stop did not end the preview")
	}
	a.Preview("siren") // unknown: nothing plays
	f.quiet(t, 50*time.Millisecond)
	a.Close()
}

// The time-up alarm always wins: it ends a preview, and a preview cannot replace it
// (D-16).
func TestSoundAlarmWinsOverPreview(t *testing.T) {
	f, cfg := newSoundOS(t, 0)
	cfg.Sound = Rising
	a := New(cfg)
	a.Preview(Bell)
	if s := f.next(t); s != Bell {
		t.Fatalf("preview played %s", s)
	}
	a.Start()
	if s := f.next(t); s != Rising {
		t.Fatalf("alarm played %s, want rising", s)
	}
	a.Preview(Beep)
	f.quiet(t, 50*time.Millisecond)
	a.Start() // already playing: no restart
	f.quiet(t, 50*time.Millisecond)
	a.Close()
	if f.kills.Load() != 2 {
		t.Errorf("%d kills, want preview and alarm", f.kills.Load())
	}
}

func TestSoundPreviewBellWithoutPlayer(t *testing.T) {
	f := newFakeOS()
	a := New(f.config(t, "linux"))
	a.Preview(Bell)
	waitBells(t, f, 1)
	time.Sleep(30 * time.Millisecond)
	a.Close()
	if n := f.bells.Load(); n != 1 {
		t.Errorf("bell rang %d times for a preview, want once", n)
	}
}
