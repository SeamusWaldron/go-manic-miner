package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"manicminer/audio"
)

// Recorder captures gameplay video (raw RGBA frames piped to ffmpeg) and
// audio (float32 stereo samples written to a WAV file) in parallel, then
// muxes them with a second ffmpeg call when stopped.
//
// The two streams are written independently while recording so a stall on
// one (e.g. ffmpeg backpressure) can't block the other. Time alignment is
// implicit — both start writing when Start returns and stop when Stop is
// called, and ffmpeg trims to the shorter at mux time via -shortest.
type Recorder struct {
	mu sync.Mutex

	active bool

	// Video pipeline: ffmpeg consumes raw RGBA on stdin and writes an mp4.
	videoCmd   *exec.Cmd
	videoStdin io.WriteCloser

	// Audio: WAV file with a placeholder header that's rewritten at Stop.
	audioFile  *os.File
	audioBytes int64

	audioPath string
	videoPath string
	finalPath string
	gifPath   string
}

// Start opens the video and audio sinks. width/height is the source frame
// size (typically 256×192 for native Manic Miner). fps is the video frame
// rate (typically 60 — Ebitengine's display rate). Returns an error if
// ffmpeg is not on PATH or the pipes can't be opened.
func (r *Recorder) Start(width, height, fps int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active {
		return errors.New("recorder already running")
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return errors.New("ffmpeg not found in PATH; recording disabled for this session")
	}

	const recDir = "recordings"
	if err := os.MkdirAll(recDir, 0755); err != nil {
		return fmt.Errorf("create recordings dir: %w", err)
	}

	stamp := time.Now().Format("20060102_150405")
	r.audioPath = fmt.Sprintf("%s/manicminer_%s_audio.wav", recDir, stamp)
	r.videoPath = fmt.Sprintf("%s/manicminer_%s_video.mp4", recDir, stamp)
	r.finalPath = fmt.Sprintf("%s/manicminer_%s.mp4", recDir, stamp)
	r.gifPath = fmt.Sprintf("%s/manicminer_%s.gif", recDir, stamp)

	// Encode at native resolution then nearest-neighbour upscale 4× inside
	// ffmpeg so the pixel art stays crisp and the output is reasonably sized.
	upW := width * 4
	upH := height * 4
	cmd := exec.Command("ffmpeg",
		"-y",
		"-loglevel", "error",
		"-f", "rawvideo",
		"-pixel_format", "rgba",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", "-",
		"-vf", fmt.Sprintf("scale=%d:%d:flags=neighbor", upW, upH),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		r.videoPath,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdin: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start: %w", err)
	}

	audioFile, err := os.Create(r.audioPath)
	if err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		return fmt.Errorf("audio file: %w", err)
	}
	// Reserve 44 bytes for the WAV header; we'll backfill it at Stop.
	if _, err := audioFile.Write(make([]byte, 44)); err != nil {
		_ = audioFile.Close()
		_ = stdin.Close()
		_ = cmd.Wait()
		return fmt.Errorf("audio header: %w", err)
	}

	r.videoCmd = cmd
	r.videoStdin = stdin
	r.audioFile = audioFile
	r.audioBytes = 0
	r.active = true

	fmt.Printf("Recording started → %s\n", r.finalPath)
	return nil
}

// IsActive reports whether a recording is in progress.
func (r *Recorder) IsActive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// WriteFrame pipes one frame of raw RGBA pixel bytes into ffmpeg's stdin.
// Called from the render loop; safe to call when no recording is active.
func (r *Recorder) WriteFrame(rgba []byte) {
	r.mu.Lock()
	if !r.active || r.videoStdin == nil {
		r.mu.Unlock()
		return
	}
	w := r.videoStdin
	r.mu.Unlock()
	_, _ = w.Write(rgba)
}

// Write satisfies io.Writer for the audio tap installed via
// audio.Player.SetRecorder. Called from oto's audio thread.
func (r *Recorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active || r.audioFile == nil {
		return len(p), nil
	}
	n, err := r.audioFile.Write(p)
	if err == nil {
		r.audioBytes += int64(n)
	}
	return n, err
}

// Stop closes both sinks, waits for the video encoder, finalises the WAV
// header, and runs a final ffmpeg call to mux the two intermediates into
// a single mp4. The intermediates are deleted on success.
func (r *Recorder) Stop() {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return
	}
	r.active = false

	stdin := r.videoStdin
	cmd := r.videoCmd
	audioFile := r.audioFile
	audioBytes := r.audioBytes
	audioPath := r.audioPath
	videoPath := r.videoPath
	finalPath := r.finalPath
	gifPath := r.gifPath

	r.videoStdin = nil
	r.videoCmd = nil
	r.audioFile = nil
	r.mu.Unlock()

	// Close video stdin so ffmpeg flushes and exits.
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil {
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Recording: ffmpeg video encode error: %v\n", err)
		}
	}

	// Backfill the WAV header now that we know the data length.
	if audioFile != nil {
		writeWAVHeader(audioFile, audioBytes)
		_ = audioFile.Close()
	}

	// Mux video + audio.
	muxCmd := exec.Command("ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", videoPath,
		"-i", audioPath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-shortest",
		finalPath,
	)
	muxCmd.Stderr = os.Stderr
	if err := muxCmd.Run(); err != nil {
		fmt.Printf("Recording: mux error: %v (intermediates kept: %s, %s)\n", err, videoPath, audioPath)
		return
	}

	_ = os.Remove(audioPath)
	_ = os.Remove(videoPath)

	fmt.Printf("Recording saved: %s\n", finalPath)

	// Generate an animated GIF alongside the mp4 for posting to GitHub /
	// Reddit. Two-pass palette generation gives clean colours without
	// dithering blur, nearest-neighbour scaling preserves the pixel art.
	gifFilter := "fps=20,scale=512:-1:flags=neighbor,split[s0][s1];" +
		"[s0]palettegen=stats_mode=diff[p];" +
		"[s1][p]paletteuse=dither=none"
	gifCmd := exec.Command("ffmpeg",
		"-y",
		"-loglevel", "error",
		"-i", finalPath,
		"-vf", gifFilter,
		gifPath,
	)
	gifCmd.Stderr = os.Stderr
	if err := gifCmd.Run(); err != nil {
		fmt.Printf("Recording: gif generation error: %v (mp4 still saved at %s)\n", err, finalPath)
		return
	}
	fmt.Printf("Animated GIF saved: %s\n", gifPath)
}

// writeWAVHeader rewrites bytes 0–43 of the WAV file with a valid RIFF
// header for IEEE float32 stereo at the audio package's sample rate, given
// the size of the data payload that follows.
func writeWAVHeader(f *os.File, dataBytes int64) {
	const (
		channels      = uint16(2)
		bitsPerSample = uint16(32)
	)
	sampleRate := uint32(audio.SampleRate)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	blockAlign := channels * bitsPerSample / 8
	fileSize := uint32(dataBytes + 36)

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return
	}

	// RIFF chunk.
	_, _ = f.Write([]byte("RIFF"))
	_ = binary.Write(f, binary.LittleEndian, fileSize)
	_, _ = f.Write([]byte("WAVE"))

	// fmt sub-chunk (16 bytes, format 3 = IEEE float).
	_, _ = f.Write([]byte("fmt "))
	_ = binary.Write(f, binary.LittleEndian, uint32(16))
	_ = binary.Write(f, binary.LittleEndian, uint16(3))
	_ = binary.Write(f, binary.LittleEndian, channels)
	_ = binary.Write(f, binary.LittleEndian, sampleRate)
	_ = binary.Write(f, binary.LittleEndian, byteRate)
	_ = binary.Write(f, binary.LittleEndian, blockAlign)
	_ = binary.Write(f, binary.LittleEndian, bitsPerSample)

	// data sub-chunk header.
	_, _ = f.Write([]byte("data"))
	_ = binary.Write(f, binary.LittleEndian, uint32(dataBytes))
}
