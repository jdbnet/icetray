package player

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"

	"git.jdbnet.co.uk/jamie/icetray/logger"
)

// StreamBuffer interface allows player to consume RingBuffer without direct package dependency.
type StreamBuffer interface {
	io.ReadCloser
	AvailableData() int
	IsClosed() bool
}

// Player manages the beep-based audio decoding and playback lifecycle.
type Player struct {
	mu           sync.RWMutex
	volume       int
	isRunning    bool
	isPaused     bool
	activeBuf    StreamBuffer
	activeCancel chan struct{}
	ctrl         *beep.Ctrl
	volumeEffect *effects.Volume
}

// NewPlayer creates a new Player instance and initialises the speaker once.
func NewPlayer() *Player {
	p := &Player{
		volume:    100, // Default volume
		isRunning: false,
	}

	// Initialize speaker once at startup
	sr := beep.SampleRate(44100)
	// buffer size of 1/10s (4410 samples)
	err := speaker.Init(sr, sr.N(time.Second/10))
	if err != nil {
		logger.LogError("Failed to initialize speaker", err)
	} else {
		logger.Log("Speaker initialized successfully at 44100Hz")
	}

	return p
}

// Play starts playing an audio stream (sets state).
func (p *Player) Play(streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("player is already running")
	}

	p.isRunning = true
	p.isPaused = false
	logger.Log("Play: player running state set for stream " + streamURL)
	return nil
}

// SetSource starts playback from a new buffer source.
func (p *Player) SetSource(buf StreamBuffer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		// If the player is not running, ignore the source
		return
	}

	p.stopActiveStream()

	p.activeBuf = buf
	p.activeCancel = make(chan struct{})
	go p.playSource(buf, p.activeCancel)
}

// ClearSource stops any active playback stream but keeps running state.
func (p *Player) ClearSource() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopActiveStream()
}

// stopActiveStream stops the active streamer and cancels the goroutine. Must be called with p.mu locked.
func (p *Player) stopActiveStream() {
	if p.activeCancel != nil {
		close(p.activeCancel)
		p.activeCancel = nil
	}
	if p.activeBuf != nil {
		p.activeBuf.Close()
		p.activeBuf = nil
	}
	speaker.Clear()
	speaker.Lock()
	if p.ctrl != nil {
		p.ctrl.Streamer = nil
		p.ctrl = nil
	}
	p.volumeEffect = nil
	speaker.Unlock()
}

// playSource decodes and plays the audio source.
func (p *Player) playSource(buf StreamBuffer, cancel chan struct{}) {
	// 1. Wait for buffer to fill (target ~10 seconds of buffer).
	// Target is 200 KB (about 10 seconds at 160kbps).
	targetBytes := 200 * 1024
	logger.Log(fmt.Sprintf("playSource: waiting for buffer to fill to %d bytes...", targetBytes))

	for {
		select {
		case <-cancel:
			return
		default:
		}

		if buf.AvailableData() >= targetBytes {
			break
		}

		if buf.IsClosed() {
			logger.Log("playSource: buffer closed before reaching target size")
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	select {
	case <-cancel:
		return
	default:
	}

	logger.Log(fmt.Sprintf("playSource: buffer filled (%d bytes), decoding...", buf.AvailableData()))

	// 2. Decode the MP3 stream
	streamer, format, err := mp3.Decode(buf)
	if err != nil {
		logger.LogError("playSource: failed to decode stream", err)
		return
	}
	defer streamer.Close()

	// 3. Setup volume effect
	p.mu.Lock()
	vol := p.volume
	p.mu.Unlock()

	volumeEffect := &effects.Volume{
		Streamer: streamer,
		Base:     2.0,
	}
	if vol == 0 {
		volumeEffect.Volume = -10.0
		volumeEffect.Silent = true
	} else {
		volumeEffect.Volume = (float64(vol) - 100.0) / 10.0
		volumeEffect.Silent = false
	}

	// 4. Resample stream to speaker's sample rate (44100Hz)
	resampled := beep.Resample(4, format.SampleRate, beep.SampleRate(44100), volumeEffect)

	// 5. Wrap in Ctrl to support Pause/Resume
	p.mu.Lock()
	isPaused := p.isPaused
	ctrl := &beep.Ctrl{
		Streamer: resampled,
		Paused:   isPaused,
	}
	p.ctrl = ctrl
	p.volumeEffect = volumeEffect
	p.mu.Unlock()

	// 6. Play on speaker
	speaker.Play(ctrl)
	logger.Log("playSource: speaker playback started")

	// 7. Wait until finished or cancelled
	<-cancel
	logger.Log("playSource: playback goroutine exiting")
}

// Pause pauses the playback.
func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("player not running")
	}
	if p.isPaused {
		return nil
	}

	p.isPaused = true
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
	logger.Log("Pause: playback paused")
	return nil
}

// Resume resumes playback.
func (p *Player) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("player not running")
	}
	if !p.isPaused {
		return nil
	}

	p.isPaused = false
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
	}
	logger.Log("Resume: playback resumed")
	return nil
}

// Stop stops the playback and cleans up active stream.
func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	p.stopActiveStream()
	p.isRunning = false
	p.isPaused = false
	logger.Log("Stop: playback stopped")
	return nil
}

// SetVolume sets the volume (0-100) using a logarithmic mapping.
func (p *Player) SetVolume(vol int) error {
	if vol < 0 {
		vol = 0
	} else if vol > 100 {
		vol = 100
	}

	p.mu.Lock()
	p.volume = vol
	volEffect := p.volumeEffect
	p.mu.Unlock()

	if volEffect != nil {
		speaker.Lock()
		if vol == 0 {
			volEffect.Volume = -10.0
			volEffect.Silent = true
		} else {
			volEffect.Volume = (float64(vol) - 100.0) / 10.0
			volEffect.Silent = false
		}
		speaker.Unlock()
	}

	logger.Log(fmt.Sprintf("Volume: set to %d", vol))
	return nil
}

// IsRunning returns whether the player is currently running.
func (p *Player) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// Close stops the player and cleans up resources.
func (p *Player) Close() error {
	p.Stop()
	return nil
}
