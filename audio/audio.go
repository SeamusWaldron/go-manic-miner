// Package audio provides ZX Spectrum beeper sound emulation using Ebitengine audio.
package audio

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	sampleRate = 44100
	// The Spectrum's CPU runs at 3.5MHz. The beeper frequency is determined by
	// a delay loop counter. Approximate: freq ≈ 3500000 / (4 * counter).
	spectrumClock = 3500000.0
)

// Player manages audio output for the game.
type Player struct {
	context    *audio.Context
	tonePlayer *audio.Player
	stream     *toneStream
}

// NewPlayer creates a new audio Player.
func NewPlayer() *Player {
	ctx := audio.NewContext(sampleRate)
	stream := &toneStream{
		sampleRate: sampleRate,
	}
	player, _ := ctx.NewPlayerF32(stream)
	player.SetBufferSize(sampleRate / 10 * 4 * 2) // ~100ms buffer, stereo float32.
	player.Play()

	return &Player{
		context:    ctx,
		tonePlayer: player,
		stream:     stream,
	}
}

// PlayNote sets the current tone. freq1 and freq2 are Spectrum frequency
// parameters (delay loop counters). Set both to 0 to silence.
func (p *Player) PlayNote(freq1, freq2 byte) {
	if freq1 == 0 && freq2 == 0 {
		p.stream.setFrequencies(0, 0)
		return
	}

	// Convert Spectrum delay counter to Hz.
	// The beeper toggles every (counter * 4) T-states.
	// Full cycle = counter * 8 T-states. Freq = clock / (counter * 8).
	var hz1, hz2 float64
	if freq1 > 0 {
		hz1 = spectrumClock / (float64(freq1) * 8.0)
	}
	if freq2 > 0 {
		hz2 = spectrumClock / (float64(freq2) * 8.0)
	}

	p.stream.setFrequencies(hz1, hz2)
}

// PlayInGameNote plays a single in-game music note (Mountain King).
func (p *Player) PlayInGameNote(freq byte) {
	if freq == 0 {
		p.stream.setFrequencies(0, 0)
		return
	}
	hz := spectrumClock / (float64(freq) * 8.0)
	p.stream.setFrequencies(hz, 0)
}

// Silence stops all audio output.
func (p *Player) Silence() {
	p.stream.setFrequencies(0, 0)
}

// toneStream generates a square wave at one or two frequencies.
type toneStream struct {
	sampleRate int
	phase1     float64
	phase2     float64
	freq1      float64
	freq2      float64
}

func (s *toneStream) setFrequencies(f1, f2 float64) {
	s.freq1 = f1
	s.freq2 = f2
}

// Read implements io.Reader for Ebitengine audio (stereo float32).
func (s *toneStream) Read(buf []byte) (int, error) {
	// buf is []byte but represents []float32 (4 bytes per sample, stereo = 2 channels).
	numFloat32s := len(buf) / 4
	numSamples := numFloat32s / 2 // Stereo pairs.

	for i := 0; i < numSamples; i++ {
		var sample float32

		if s.freq1 > 0 {
			// Square wave: +0.15 or -0.15.
			if math.Mod(s.phase1, 1.0) < 0.5 {
				sample += 0.15
			} else {
				sample -= 0.15
			}
			s.phase1 += s.freq1 / float64(s.sampleRate)
		}

		if s.freq2 > 0 {
			if math.Mod(s.phase2, 1.0) < 0.5 {
				sample += 0.10
			} else {
				sample -= 0.10
			}
			s.phase2 += s.freq2 / float64(s.sampleRate)
		}

		// Write stereo float32 (little-endian).
		bits := math.Float32bits(sample)
		offset := i * 8 // 2 channels * 4 bytes.
		// Left channel.
		buf[offset+0] = byte(bits)
		buf[offset+1] = byte(bits >> 8)
		buf[offset+2] = byte(bits >> 16)
		buf[offset+3] = byte(bits >> 24)
		// Right channel (same).
		buf[offset+4] = byte(bits)
		buf[offset+5] = byte(bits >> 8)
		buf[offset+6] = byte(bits >> 16)
		buf[offset+7] = byte(bits >> 24)
	}

	return numSamples * 8, nil
}
