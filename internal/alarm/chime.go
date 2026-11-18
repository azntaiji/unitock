package alarm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

// ChimeFile is the alarm sound's name inside the data directory (D-13).
const ChimeFile = "alarm.wav"

const (
	sampleRate  = 22050
	chimeLength = 1.5 // seconds, including the quiet tail that spaces out the loop
)

// chime synthesises the alarm: two bell-like notes, a fifth apart, each a sine with a
// quieter octave that fades out, as 16-bit mono PCM WAV (D-13).
func chime() []byte {
	notes := []struct{ at, freq float64 }{{0, 880}, {0.25, 1318.5}}
	n := int(chimeLength * sampleRate)
	pcm := make([]int16, n)
	for i := range pcm {
		t := float64(i) / sampleRate
		var v float64
		for _, nt := range notes {
			if d := t - nt.at; d >= 0 {
				env := math.Exp(-d / 0.3)
				v += env * (math.Sin(2*math.Pi*nt.freq*d) + 0.3*math.Sin(4*math.Pi*nt.freq*d))
			}
		}
		pcm[i] = int16(math.Max(-1, math.Min(1, 0.35*v)) * math.MaxInt16)
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

// writeChime writes the chime to dir unless an identical file is already there, and
// returns its path.
func writeChime(dir string) (string, error) {
	path := filepath.Join(dir, ChimeFile)
	want := chime()
	if got, err := os.ReadFile(path); err == nil && bytes.Equal(got, want) {
		return path, nil
	}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		return "", fmt.Errorf("alarm sound: %w", err)
	}
	return path, nil
}
