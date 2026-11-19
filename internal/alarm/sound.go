package alarm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// ChimeFile is the alarm sound's name inside the data directory (D-13). It holds the
// sound last played, rewritten when that changes (D-16).
const ChimeFile = "alarm.wav"

// Sound is an alarm sound the user can choose (R-51, D-16). Its value is what the store
// keeps.
type Sound string

// The alarm sounds, all synthesised here (D-13, D-16).
const (
	Chime   Sound = "chime"
	Bell    Sound = "bell"
	Beep    Sound = "beep"
	Marimba Sound = "marimba"
	Rising  Sound = "rising"
)

// DefaultSound is used until the user chooses one, and for a missing or unknown stored
// value (R-53).
const DefaultSound = Chime

// Sounds lists every sound in the order the Sound screen shows them.
var Sounds = []Sound{Chime, Bell, Beep, Marimba, Rising}

// ParseSound returns the sound named s, or false if there is none.
func ParseSound(s string) (Sound, bool) {
	for _, snd := range Sounds {
		if string(snd) == s {
			return snd, true
		}
	}
	return "", false
}

// Label is the sound's name for display.
func (s Sound) Label() string {
	switch s {
	case Chime:
		return "Chime"
	case Bell:
		return "Bell"
	case Beep:
		return "Beep"
	case Marimba:
		return "Marimba"
	case Rising:
		return "Rising"
	}
	return string(s)
}

const sampleRate = 22050

// targetPeak is the loudest sample of every sound but Chime, as a fraction of full
// scale, so they play at similar levels. Chime keeps its original level (D-13).
const targetPeak = 0.5

// voice describes one sound: its length in seconds, including the quiet tail that
// spaces out the loop, and its waveform at time t seconds.
type voice struct {
	length float64
	wave   func(t float64) float64
}

// partial is one sine component of a struck note: its frequency as a ratio of the
// note's, amplitude, and exponential decay time constant in seconds.
type partial struct{ ratio, amp, tau float64 }

// strike is the sound at d seconds after striking a note of freq with partials, with a
// short linear attack so the onset does not click. It is silent before the strike.
func strike(d, freq, attack float64, ps []partial) float64 {
	if d < 0 {
		return 0
	}
	var v float64
	for _, p := range ps {
		v += p.amp * math.Exp(-d/p.tau) * math.Sin(2*math.Pi*freq*p.ratio*d)
	}
	if attack > 0 && d < attack {
		v *= d / attack
	}
	return v
}

var voices = map[Sound]voice{
	// Two bell-like notes a fifth apart, each a sine with a quieter octave (D-13).
	Chime: {1.5, func(t float64) float64 {
		var v float64
		for _, n := range []struct{ at, freq float64 }{{0, 880}, {0.25, 1318.5}} {
			if d := t - n.at; d >= 0 {
				env := math.Exp(-d / 0.3)
				v += env * (math.Sin(2*math.Pi*n.freq*d) + 0.3*math.Sin(4*math.Pi*n.freq*d))
			}
		}
		return v
	}},
	// One long strike with a church bell's inharmonic partials: hum, prime, minor
	// third, fifth, nominal, and a bright upper partial.
	Bell: {2.2, func(t float64) float64 {
		return strike(t, 660, 0.003, []partial{
			{0.5, 0.35, 1.2}, {1, 1, 0.9}, {1.19, 0.4, 0.6}, {1.5, 0.3, 0.5}, {2, 0.35, 0.4}, {2.74, 0.2, 0.25},
		})
	}},
	// Four short, even pulses, like a digital alarm clock.
	Beep: {1.2, func(t float64) float64 {
		const on, gap, ramp, freq = 0.1, 0.05, 0.005, 1760
		for i := range 4 {
			d := t - float64(i)*(on+gap)
			if d < 0 || d >= on {
				continue
			}
			env := min(1, d/ramp, (on-d)/ramp)
			return env * (math.Sin(2*math.Pi*freq*d) + 0.25*math.Sin(6*math.Pi*freq*d))
		}
		return 0
	}},
	// Three quick wooden notes, down and back up: a fundamental that dies fast and the
	// fourth harmonic a marimba bar is tuned to, dying faster.
	Marimba: {1.5, func(t float64) float64 {
		ps := []partial{{1, 1, 0.13}, {4, 0.3, 0.035}}
		return strike(t, 1046.5, 0.002, ps) + strike(t-0.16, 784, 0.002, ps) + strike(t-0.32, 1046.5, 0.002, ps)
	}},
	// A rising major triad, C5, E5, G5, each note ringing over the next.
	Rising: {1.8, func(t float64) float64 {
		ps := []partial{{1, 1, 0.35}, {2, 0.2, 0.2}}
		return strike(t, 523.25, 0.005, ps) + strike(t-0.25, 659.25, 0.005, ps) + strike(t-0.5, 783.99, 0.005, ps)
	}},
}

// WAV returns the sound as 16-bit mono PCM WAV (D-13). An unknown sound plays Chime.
func (s Sound) WAV() []byte {
	vc, ok := voices[s]
	if !ok {
		s = DefaultSound
		vc = voices[s]
	}
	n := int(vc.length * sampleRate)
	samples := make([]float64, n)
	var peak float64
	for i := range samples {
		samples[i] = vc.wave(float64(i) / sampleRate)
		peak = max(peak, math.Abs(samples[i]))
	}
	gain := 0.35 // Chime's original level
	if s != Chime && peak > 0 {
		gain = targetPeak / peak
	}
	pcm := make([]int16, n)
	for i, v := range samples {
		pcm[i] = int16(math.Max(-1, math.Min(1, gain*v)) * math.MaxInt16)
	}

	var b bytes.Buffer
	dataLen := uint32(2 * n)
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(36 + dataLen)
	b.WriteString("WAVEfmt ")
	w(uint32(16))         // fmt chunk size
	w(uint16(1))          // PCM
	w(uint16(1))          // mono
	w(uint32(sampleRate)) // sample rate
	w(uint32(2 * sampleRate))
	w(uint16(2))  // block align
	w(uint16(16)) // bits per sample
	b.WriteString("data")
	w(dataLen)
	w(pcm)
	return b.Bytes()
}

// writeSound writes s to dir as ChimeFile unless an identical file is already there,
// and returns its path.
func writeSound(dir string, s Sound) (string, error) {
	path := filepath.Join(dir, ChimeFile)
	want := s.WAV()
	if got, err := os.ReadFile(path); err == nil && bytes.Equal(got, want) {
		return path, nil
	}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		return "", fmt.Errorf("alarm sound: %w", err)
	}
	return path, nil
}
