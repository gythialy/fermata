// Command gensounds writes the app's four built-in sounds as 16-bit mono WAV
// files. They are original tones, not recordings.
//
//	go run ./tools/gensounds
package main

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
)

const (
	sampleRate = 44100
	outDir     = "internal/sounds/assets"
)

func main() {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	sounds := map[string][]float32{
		"crystal-glass.wav": crystalGlass(),
		"wind-chime.wav":    windChime(),
		"tic-toc.wav":       ticToc(),
		"reverie.wav":       reverie(),
	}
	for name, samples := range sounds {
		if err := writeWAV(filepath.Join(outDir, name), samples); err != nil {
			log.Fatal(err)
		}
	}
}

// crystalGlass is a struck glass: a bright tone and two overtones, each dying
// away at its own pace.
func crystalGlass() []float32 {
	const duration = 1.4
	out := make([]float32, int(duration*sampleRate))
	for i := range out {
		t := float64(i) / sampleRate
		envelope := math.Exp(-t / 0.32)
		v := 0.55*math.Sin(2*math.Pi*1567.98*t)*envelope +
			0.25*math.Sin(2*math.Pi*2.7*1567.98*t)*math.Exp(-t/0.12) +
			0.12*math.Sin(2*math.Pi*5.1*1567.98*t)*math.Exp(-t/0.05)
		out[i] = float32(v)
	}
	return fade(out, 0.004, 0.05)
}

// windChime is a handful of detuned tubes, struck one after another.
func windChime() []float32 {
	const duration = 2.2
	out := make([]float32, int(duration*sampleRate))
	partials := []struct {
		freq, gain, decay, start float64
	}{
		{880, 0.30, 0.55, 0.00},
		{1174.66, 0.26, 0.60, 0.09},
		{1396.91, 0.22, 0.50, 0.18},
		{1760, 0.18, 0.45, 0.27},
		{2093, 0.12, 0.35, 0.36},
	}
	for i := range out {
		t := float64(i) / sampleRate
		var v float64
		for _, p := range partials {
			if t < p.start {
				continue
			}
			dt := t - p.start
			v += p.gain * math.Sin(2*math.Pi*p.freq*dt) * math.Exp(-dt/p.decay)
			v += 0.3 * p.gain * math.Sin(2*math.Pi*p.freq*2.01*dt) * math.Exp(-dt/p.decay/2)
		}
		out[i] = float32(v)
	}
	return fade(out, 0.003, 0.1)
}

// ticToc is two small clicks, like a clock's, far apart.
func ticToc() []float32 {
	const duration = 1.1
	out := make([]float32, int(duration*sampleRate))
	rng := rand.New(rand.NewPCG(7, 11))
	click := func(at float64, freq float64) {
		start := int(at * sampleRate)
		for i := 0; i < int(0.05*sampleRate) && start+i < len(out); i++ {
			dt := float64(i) / sampleRate
			noise := rng.Float64()*2 - 1
			v := 0.5*noise*math.Exp(-dt/0.002) +
				0.4*math.Sin(2*math.Pi*freq*dt)*math.Exp(-dt/0.012)
			out[start+i] += float32(v)
		}
	}
	click(0.02, 1800)
	click(0.62, 1400)
	return fade(out, 0.001, 0.02)
}

// reverie is a soft chord that swells and settles.
func reverie() []float32 {
	const duration = 2.6
	out := make([]float32, int(duration*sampleRate))
	notes := []float64{220, 261.63, 329.63, 440}
	for i := range out {
		t := float64(i) / sampleRate
		attack := 1 - math.Exp(-t/0.35)
		release := math.Exp(-math.Max(t-0.9, 0) / 0.7)
		var v float64
		for k, f := range notes {
			v += (0.20 - 0.03*float64(k)) * math.Sin(2*math.Pi*f*t)
		}
		out[i] = float32(v * attack * release)
	}
	return fade(out, 0.02, 0.25)
}

// fade applies a short fade in and out so that a sound never clicks.
func fade(samples []float32, in, out float64) []float32 {
	inN := min(int(in*sampleRate), len(samples))
	outN := min(int(out*sampleRate), len(samples))
	for i := 0; i < inN; i++ {
		samples[i] *= float32(i) / float32(max(inN, 1))
	}
	for i := 0; i < outN; i++ {
		samples[len(samples)-1-i] *= float32(i) / float32(max(outN, 1))
	}
	peak := 0.0
	for _, s := range samples {
		peak = max(peak, math.Abs(float64(s)))
	}
	if peak > 0.8 {
		scale := 0.8 / peak
		for i := range samples {
			samples[i] = float32(float64(samples[i]) * scale)
		}
	}
	return samples
}

// writeWAV writes mono 16-bit PCM with a canonical 44-byte header.
func writeWAV(path string, samples []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dataSize := len(samples) * 2
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+dataSize))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:], 1) // mono
	binary.LittleEndian.PutUint32(header[24:], sampleRate)
	binary.LittleEndian.PutUint32(header[28:], sampleRate*2)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(dataSize))
	if _, err := f.Write(header); err != nil {
		return err
	}
	buf := make([]byte, 2)
	for _, s := range samples {
		v := int16(math.Round(min(max(float64(s), -1), 1) * 32767))
		binary.LittleEndian.PutUint16(buf, uint16(v))
		if _, err := f.Write(buf); err != nil {
			return err
		}
	}
	return nil
}
