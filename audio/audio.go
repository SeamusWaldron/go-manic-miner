// Package audio provides ZX Spectrum beeper sound emulation using Ebitengine audio.
package audio

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	sampleRate    = 44100
	spectrumClock = 3500000.0
	volume        = 0.35 // Louder to compensate for short burst duty cycle.
)

// Player manages audio output for the game.
type Player struct {
	context *audio.Context
	player  *audio.Player
	stream  *toneStream
}

// NewPlayer creates a new audio Player.
func NewPlayer() *Player {
	ctx := audio.NewContext(sampleRate)
	stream := newToneStream()
	player, _ := ctx.NewPlayerF32(stream)
	player.Play()

	return &Player{
		context: ctx,
		player:  player,
		stream:  stream,
	}
}

// PlayTune starts playing the title tune (Blue Danube). The tune plays at
// the correct tempo internally — the caller doesn't need to step through notes.
// tuneData is the raw note data (3 bytes per note: duration, freq1, freq2, terminated by $FF).
func (p *Player) PlayTune(tuneData []byte) {
	p.stream.startTune(tuneData)
}

// TuneNoteIndex returns the index of the note currently being played (for piano key animation).
func (p *Player) TuneNoteIndex() int {
	p.stream.mu.Lock()
	defer p.stream.mu.Unlock()
	return p.stream.tuneNoteIdx
}

// IsTunePlaying returns true if the title tune is still playing.
func (p *Player) IsTunePlaying() bool {
	p.stream.mu.Lock()
	defer p.stream.mu.Unlock()
	return p.stream.tunePlaying
}

// PlayInGameNote plays a single in-game music note as a short burst.
// PlayInGameNote plays a short burst matching the original's ~8.8ms duration.
// The original plays 768 iterations of a ~40 T-state loop = 30,720 T = 8.8ms.
// This staccato articulation is what makes the tune sound fast and energetic.
func (p *Player) PlayInGameNote(freq byte) {
	if freq == 0 {
		p.stream.setTone(0, 0)
		return
	}
	hz := spectrumClock / (float64(freq) * 80.0)
	// 8.8ms at 44100 Hz = 388 samples. Use 600 samples (~14ms) for slightly
	// better pitch audibility while keeping the staccato character.
	p.stream.playBurst(hz, 600)
}

// PlaySFX plays a short sound effect (jump/fall). Pitch is the D parameter
// from the original Z80 code. The sound plays as a short burst.
func (p *Player) PlaySFX(pitch int) {
	if pitch <= 0 {
		return
	}
	// Convert D parameter to Hz. Original loop: OUT, XOR, LD B,D, DJNZ.
	// Half period = D * 13 T-states (DJNZ loop). Full cycle = D * 26 T.
	// Hz = 3500000 / (D * 26).
	hz := spectrumClock / (float64(pitch) * 26.0)
	// Duration: C=32 outer loops, each D inner loops.
	// Total = 32 * (13*D + 33) T-states. Convert to samples.
	totalT := 32.0 * (13.0*float64(pitch) + 33.0)
	durSecs := totalT / spectrumClock
	dur := int(durSecs * float64(sampleRate))
	if dur < 100 {
		dur = 100
	}
	p.stream.playBurst(hz, dur)
}

// Silence stops all audio output.
func (p *Player) Silence() {
	p.stream.stopTune()
	p.stream.setTone(0, 0)
}

// toneStream generates square wave tones and handles tune playback.
// Thread-safe: control methods called from game goroutine,
// Read called from audio goroutine.
type toneStream struct {
	mu     sync.Mutex
	freq1  float64
	freq2  float64
	phase1 float64
	phase2 float64

	// Tune playback state (managed internally by Read).
	tunePlaying     bool
	tuneData        []byte
	tuneNoteIdx     int
	tuneSamplesLeft int // Samples remaining for current note.

	// Burst mode: play a tone for a fixed number of samples, then silence.
	burstSamplesLeft int
}

func newToneStream() *toneStream {
	return &toneStream{}
}

func (s *toneStream) setTone(f1, f2 float64) {
	s.mu.Lock()
	s.freq1 = f1
	s.freq2 = f2
	s.burstSamplesLeft = -1 // Continuous mode.
	s.mu.Unlock()
}

func (s *toneStream) playBurst(hz float64, samples int) {
	s.mu.Lock()
	s.freq1 = hz
	s.freq2 = 0
	s.burstSamplesLeft = samples
	s.mu.Unlock()
}

func (s *toneStream) startTune(data []byte) {
	s.mu.Lock()
	s.tuneData = data
	s.tuneNoteIdx = 0
	s.tuneSamplesLeft = 0
	s.tunePlaying = true
	s.loadNextTuneNote()
	s.mu.Unlock()
}

func (s *toneStream) stopTune() {
	s.mu.Lock()
	s.tunePlaying = false
	s.freq1 = 0
	s.freq2 = 0
	s.tuneSamplesLeft = 0
	s.mu.Unlock()
}

// loadNextTuneNote advances to the next note in the tune. Must hold mu.
func (s *toneStream) loadNextTuneNote() {
	offset := s.tuneNoteIdx * 3
	if offset+2 >= len(s.tuneData) || s.tuneData[offset] == 0xFF {
		s.tunePlaying = false
		s.freq1 = 0
		s.freq2 = 0
		return
	}

	duration := s.tuneData[offset]
	freq1 := s.tuneData[offset+1]
	freq2 := s.tuneData[offset+2]

	// Convert frequency params to Hz.
	// Title tune loop: ~56 T-states per inner iteration, 256 inner iterations.
	// Note duration = duration * 256 * 56 / 3500000 seconds.
	// In samples: duration * 256 * 56 / 3500000 * 44100.
	noteDurationSecs := float64(duration) * 256.0 * 56.0 / spectrumClock
	s.tuneSamplesLeft = int(noteDurationSecs * float64(sampleRate))

	if freq1 > 0 {
		s.freq1 = spectrumClock / (float64(freq1) * 112.0)
	} else {
		s.freq1 = 0
	}
	if freq2 > 0 {
		s.freq2 = spectrumClock / (float64(freq2) * 112.0)
	} else {
		s.freq2 = 0
	}
}

// Read fills buf with stereo float32 samples.
func (s *toneStream) Read(buf []byte) (int, error) {
	bytesPerSample := 8 // 2 channels × 4 bytes per float32.
	numSamples := len(buf) / bytesPerSample
	written := 0

	s.mu.Lock()
	f1 := s.freq1
	f2 := s.freq2
	playing := s.tunePlaying
	burst := s.burstSamplesLeft
	s.mu.Unlock()

	for i := 0; i < numSamples; i++ {
		// If playing a tune, check if current note has expired.
		if playing {
			s.mu.Lock()
			s.tuneSamplesLeft--
			if s.tuneSamplesLeft <= 0 {
				s.tuneNoteIdx++
				s.loadNextTuneNote()
				f1 = s.freq1
				f2 = s.freq2
				playing = s.tunePlaying
			}
			s.mu.Unlock()
		}

		// Burst mode: count down and silence when expired.
		if burst >= 0 {
			burst--
			if burst <= 0 {
				f1 = 0
				f2 = 0
			}
		}

		var sample float32
		if f1 > 0 {
			if s.phase1 < 0.5 {
				sample += volume
			} else {
				sample -= volume
			}
			s.phase1 += f1 / float64(sampleRate)
			s.phase1 -= math.Floor(s.phase1)
		}

		if f2 > 0 {
			if s.phase2 < 0.5 {
				sample += volume * 0.7
			} else {
				sample -= volume * 0.7
			}
			s.phase2 += f2 / float64(sampleRate)
			s.phase2 -= math.Floor(s.phase2)
		}

		bits := math.Float32bits(sample)
		offset := i * bytesPerSample
		buf[offset+0] = byte(bits)
		buf[offset+1] = byte(bits >> 8)
		buf[offset+2] = byte(bits >> 16)
		buf[offset+3] = byte(bits >> 24)
		buf[offset+4] = byte(bits)
		buf[offset+5] = byte(bits >> 8)
		buf[offset+6] = byte(bits >> 16)
		buf[offset+7] = byte(bits >> 24)

		written += bytesPerSample
	}

	return written, nil
}
