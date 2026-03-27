// Package audio provides ZX Spectrum beeper sound emulation using oto directly
// (bypassing Ebitengine's audio pipeline for low latency).
package audio

import (
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	sampleRate    = 44100
	spectrumClock = 3500000.0
	volume        = 0.10
)

// Player manages audio output for the game.
type Player struct {
	ctx    *oto.Context
	player *oto.Player
	stream *toneStream
}

// NewPlayer creates a new audio Player with low-latency oto backend.
func NewPlayer() *Player {
	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
		BufferSize:   20 * time.Millisecond, // Low latency.
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		panic(err)
	}
	<-ready

	stream := newToneStream()
	player := ctx.NewPlayer(stream)
	// The oto player defaults to a 256KB internal buffer (~740ms latency!).
	// Set it to 4096 bytes (~12ms) for near-instant response.
	player.SetBufferSize(4096)
	player.Play()

	return &Player{
		ctx:    ctx,
		player: player,
		stream: stream,
	}
}

// PlayTune starts the title tune (Blue Danube).
func (p *Player) PlayTune(tuneData []byte) {
	p.stream.startTune(tuneData)
}

// TuneNoteIndex returns the current tune note index.
func (p *Player) TuneNoteIndex() int {
	p.stream.mu.Lock()
	defer p.stream.mu.Unlock()
	return p.stream.tuneNoteIdx
}

// IsTunePlaying returns true if the title tune is playing.
func (p *Player) IsTunePlaying() bool {
	p.stream.mu.Lock()
	defer p.stream.mu.Unlock()
	return p.stream.tunePlaying
}

// StartInGameMusic begins the in-game music loop.
func (p *Player) StartInGameMusic(tuneData []byte, noteDurationMs int) {
	p.stream.startInGameMusic(tuneData, noteDurationMs)
}

// SetInGameMusicTempo changes note duration while playing.
func (p *Player) SetInGameMusicTempo(noteDurationMs int) {
	p.stream.mu.Lock()
	if noteDurationMs < 5 {
		noteDurationMs = 5
	}
	p.stream.igmNoteSamples = sampleRate * noteDurationMs / 1000
	p.stream.igmSilenceSamples = p.stream.igmNoteSamples / 2
	p.stream.mu.Unlock()
}

// IsInGameMusicPlaying returns true if in-game music is active.
func (p *Player) IsInGameMusicPlaying() bool {
	p.stream.mu.Lock()
	defer p.stream.mu.Unlock()
	return p.stream.igmPlaying
}

// StopInGameMusic stops the in-game music and silences output.
func (p *Player) StopInGameMusic() {
	p.stream.mu.Lock()
	p.stream.igmPlaying = false
	p.stream.freq1 = 0
	p.stream.freq2 = 0
	p.stream.mu.Unlock()
}

// PlaySFX plays a sound effect (jump/fall).
func (p *Player) PlaySFX(pitch int) {
	if pitch <= 0 {
		return
	}
	hz := spectrumClock / (float64(pitch) * 26.0)
	dur := sampleRate * 40 / 1000 // 40ms burst.
	p.stream.playBurst(hz, dur)
}

// Silence stops all audio immediately.
func (p *Player) Silence() {
	p.stream.mu.Lock()
	p.stream.tunePlaying = false
	p.stream.igmPlaying = false
	p.stream.freq1 = 0
	p.stream.freq2 = 0
	p.stream.burstSamplesLeft = 0
	p.stream.tuneSamplesLeft = 0
	p.stream.mu.Unlock()
}

// toneStream generates square wave tones.
type toneStream struct {
	mu     sync.Mutex
	freq1  float64
	freq2  float64
	phase1 float64
	phase2 float64

	// Title tune.
	tunePlaying     bool
	tuneData        []byte
	tuneNoteIdx     int
	tuneSamplesLeft int

	// Burst (SFX).
	burstFreq        float64
	burstSamplesLeft int

	// In-game music.
	igmPlaying        bool
	igmData           []byte
	igmNoteIdx        int
	igmNoteSamples    int
	igmSamplesLeft    int
	igmSilenceSamples int
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

func (s *toneStream) playBurst(hz float64, samples int) {
	s.mu.Lock()
	s.burstFreq = hz
	s.burstSamplesLeft = samples
	s.mu.Unlock()
}

func (s *toneStream) startInGameMusic(tuneData []byte, noteDurationMs int) {
	s.mu.Lock()
	s.igmData = tuneData
	s.igmNoteIdx = 0
	s.igmPlaying = true
	s.igmNoteSamples = sampleRate * noteDurationMs / 1000
	s.igmSilenceSamples = s.igmNoteSamples / 2
	s.igmSamplesLeft = s.igmNoteSamples
	s.tunePlaying = false
	noteFreq := tuneData[0]
	if noteFreq > 0 {
		s.freq1 = spectrumClock / (float64(noteFreq) * 80.0)
	}
	s.freq2 = 0
	s.mu.Unlock()
}

func (s *toneStream) startTune(data []byte) {
	s.mu.Lock()
	s.tuneData = data
	s.tuneNoteIdx = 0
	s.tuneSamplesLeft = 0
	s.tunePlaying = true
	s.igmPlaying = false
	s.loadNextTuneNote()
	s.mu.Unlock()
}

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
	bytesPerSample := 8
	numSamples := len(buf) / bytesPerSample
	written := 0

	s.mu.Lock()
	f1 := s.freq1
	f2 := s.freq2
	playing := s.tunePlaying
	igm := s.igmPlaying
	burst := s.burstSamplesLeft
	s.mu.Unlock()

	for i := 0; i < numSamples; i++ {
		// In-game music.
		if igm {
			s.mu.Lock()
			s.igmSamplesLeft--
			if s.igmSamplesLeft <= 0 {
				s.igmNoteIdx = (s.igmNoteIdx + 1) & 255
				noteIdx := (s.igmNoteIdx & 126) >> 1
				if noteIdx < len(s.igmData) {
					noteFreq := s.igmData[noteIdx]
					if noteFreq > 0 {
						s.freq1 = spectrumClock / (float64(noteFreq) * 80.0)
					} else {
						s.freq1 = 0
					}
				}
				s.freq2 = 0
				s.igmSamplesLeft = s.igmNoteSamples + s.igmSilenceSamples
				f1 = s.freq1
				f2 = 0
			} else if s.igmSamplesLeft <= s.igmSilenceSamples {
				f1 = 0
				f2 = 0
			} else {
				f1 = s.freq1
				f2 = 0
			}
			igm = s.igmPlaying
			s.mu.Unlock()
		}

		// Title tune.
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

		// Burst (SFX) overrides everything.
		if burst > 0 {
			burst--
			f1 = s.burstFreq
			f2 = 0
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

	s.mu.Lock()
	s.burstSamplesLeft = burst
	s.mu.Unlock()

	return written, nil
}
