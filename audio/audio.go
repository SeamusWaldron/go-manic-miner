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
	volume        = 0.20
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

// PlayInGameNote plays a single in-game music note.
func (p *Player) PlayInGameNote(freq byte) {
	if freq == 0 {
		p.stream.setTone(0, 0)
		return
	}
	// In-game loop: ~40 T-states per iteration.
	hz := spectrumClock / (float64(freq) * 80.0)
	p.stream.setTone(hz, 0)
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
}

func newToneStream() *toneStream {
	return &toneStream{}
}

func (s *toneStream) setTone(f1, f2 float64) {
	s.mu.Lock()
	s.freq1 = f1
	s.freq2 = f2
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
