package player

import (
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"

	"git.jdbnet.co.uk/jamie/icetray/logger"
)

const (
	// preBufferTargetBytes is the preferred amount of data before starting playback (~1s at 128kbps).
	preBufferTargetBytes = 16 * 1024
	// preBufferMinBytes is the minimum data required when the max wait elapses.
	preBufferMinBytes = 4 * 1024
	// preBufferMaxWait is the longest to wait for the target buffer before starting with less data.
	preBufferMaxWait = 2 * time.Second
	preBufferPollInterval = 20 * time.Millisecond
)

// uiVolumeToEffect maps a 0-100 UI level to beep's exponential Volume field.
// Gain is linear: 50% UI means 50% amplitude.
func uiVolumeToEffect(vol int) (volume float64, silent bool) {
	if vol <= 0 {
		return 0, true
	}
	gain := float64(vol) / 100.0
	return math.Log2(gain), false
}

// StreamBuffer interface allows player to consume RingBuffer without direct package dependency.
type StreamBuffer interface {
	io.ReadCloser
	AvailableData() int
	IsClosed() bool
}

// Player manages the beep-based audio decoding and playback lifecycle.
type Player struct {
	mu             sync.RWMutex
	volume         int
	speakerReady   bool
	isRunning      bool
	isPaused       bool
	audioActive    bool
	activeBuf      StreamBuffer
	activeCancel   chan struct{}
	sourceGen      uint64
	ctrl           *beep.Ctrl
	volumeEffect   *effects.Volume
	stateListeners []func()
}

// NewPlayer creates a new Player instance. Speaker output is initialised lazily on first playback.
func NewPlayer() *Player {
	return &Player{
		volume:    100,
		isRunning: false,
	}
}

// AddStateChangeListener registers a callback for playback state transitions.
func (p *Player) AddStateChangeListener(fn func()) {
	p.mu.Lock()
	p.stateListeners = append(p.stateListeners, fn)
	p.mu.Unlock()
}

func (p *Player) notifyStateChange() {
	p.mu.RLock()
	listeners := append([]func(){}, p.stateListeners...)
	p.mu.RUnlock()
	for _, fn := range listeners {
		fn()
	}
}

const speakerSampleRate = beep.SampleRate(44100)

func (p *Player) ensureSpeaker() error {
	if p.speakerReady {
		return nil
	}
	err := speaker.Init(speakerSampleRate, speakerSampleRate.N(time.Second/10))
	if err != nil {
		return err
	}
	p.speakerReady = true
	logger.Log("Speaker initialized successfully at 44100Hz")
	return nil
}

// Play starts playing an audio stream (sets intent-to-play state).
func (p *Player) Play(streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureSpeaker(); err != nil {
		return fmt.Errorf("audio output not available: %w", err)
	}

	p.isRunning = true
	p.isPaused = false
	p.audioActive = false
	logger.Log("Play: player running state set for stream " + streamURL)
	return nil
}

// SetSource starts playback from a new buffer source.
func (p *Player) SetSource(buf StreamBuffer) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return
	}

	p.stopActiveStreamLocked()
	gen := p.sourceGen
	p.activeBuf = buf
	cancel := make(chan struct{})
	p.activeCancel = cancel
	p.mu.Unlock()

	p.clearSpeaker()
	go p.playSource(buf, cancel, gen)
}

// ClearSource stops any active playback stream but keeps running state.
func (p *Player) ClearSource() {
	p.mu.Lock()
	p.stopActiveStreamLocked()
	p.mu.Unlock()
	p.clearSpeaker()
}

func (p *Player) clearSpeaker() {
	// speaker.Clear locks internally; do not wrap with speaker.Lock (self-deadlock).
	speaker.Clear()
}

// stopActiveStreamLocked stops the active streamer. Must be called with p.mu locked.
// The stream buffer is owned by the supervisor; do not close it here.
// Do not call speaker APIs here; release p.mu first to avoid deadlocks with the audio thread.
func (p *Player) stopActiveStreamLocked() {
	p.sourceGen++
	if p.activeCancel != nil {
		close(p.activeCancel)
		p.activeCancel = nil
	}
	p.activeBuf = nil
	p.ctrl = nil
	p.volumeEffect = nil

	if p.audioActive {
		p.audioActive = false
		go p.notifyStateChange()
	}
}

// playSource decodes and plays the audio source.
func (p *Player) playSource(buf StreamBuffer, cancel chan struct{}, gen uint64) {
	targetBytes := preBufferTargetBytes
	minBytes := preBufferMinBytes
	logger.Log(fmt.Sprintf("playSource: waiting for buffer to fill to %d bytes...", targetBytes))

	deadline := time.Now().Add(preBufferMaxWait)
	for {
		select {
		case <-cancel:
			return
		default:
		}

		available := buf.AvailableData()
		if available >= targetBytes {
			break
		}

		if buf.IsClosed() {
			if available < minBytes {
				logger.Log("playSource: buffer closed before reaching minimum size")
				return
			}
			logger.Log("playSource: buffer closed, starting with available data")
			break
		}

		if time.Now().After(deadline) {
			if available >= minBytes {
				logger.Log(fmt.Sprintf("playSource: max wait reached, starting with %d bytes", available))
				break
			}
			if available > 0 {
				logger.Log(fmt.Sprintf("playSource: max wait reached with %d bytes, starting anyway", available))
				break
			}
			logger.Log("playSource: max wait reached with no stream data")
			return
		}

		time.Sleep(preBufferPollInterval)
	}

	select {
	case <-cancel:
		return
	default:
	}

	logger.Log(fmt.Sprintf("playSource: buffer filled (%d bytes), decoding...", buf.AvailableData()))

	p.mu.RLock()
	stale := p.sourceGen != gen || !p.isRunning
	p.mu.RUnlock()
	if stale {
		return
	}

	synced, err := newSyncedMP3Reader(buf)
	if err != nil {
		logger.LogError("playSource: failed to find MP3 frame sync", err)
		return
	}

	// Decode the MP3 stream
	streamer, format, err := mp3.Decode(io.NopCloser(synced))
	if err != nil {
		logger.LogError("playSource: failed to decode stream", err)
		return
	}
	defer streamer.Close()

	if err := p.ensureSpeaker(); err != nil {
		logger.LogError("playSource: speaker not available", err)
		return
	}

	// 3. Setup volume effect
	p.mu.Lock()
	vol := p.volume
	p.mu.Unlock()

	volumeEffect := &effects.Volume{
		Streamer: streamer,
		Base:     2.0,
	}
	beepVol, silent := uiVolumeToEffect(vol)
	volumeEffect.Volume = beepVol
	volumeEffect.Silent = silent

	// 4. Resample to the speaker rate when needed
	var output beep.Streamer = volumeEffect
	if format.SampleRate != speakerSampleRate {
		output = beep.Resample(4, format.SampleRate, speakerSampleRate, volumeEffect)
	}

	// 5. Wrap in Ctrl to support Pause/Resume
	p.mu.Lock()
	if p.sourceGen != gen || !p.isRunning {
		p.mu.Unlock()
		return
	}
	isPaused := p.isPaused
	ctrl := &beep.Ctrl{
		Streamer: output,
		Paused:   isPaused,
	}
	p.ctrl = ctrl
	p.volumeEffect = volumeEffect
	p.mu.Unlock()

	// 6. Play on speaker (Play locks internally; do not wrap with speaker.Lock).
	speaker.Play(ctrl)
	logger.Log("playSource: speaker playback started")

	p.mu.Lock()
	p.audioActive = true
	p.mu.Unlock()
	p.notifyStateChange()

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
	go p.notifyStateChange()
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
	go p.notifyStateChange()
	return nil
}

// Stop stops the playback and cleans up active stream.
func (p *Player) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}

	p.stopActiveStreamLocked()
	p.isRunning = false
	p.isPaused = false
	p.mu.Unlock()

	p.clearSpeaker()
	logger.Log("Stop: playback stopped")
	go p.notifyStateChange()
	return nil
}

// SetVolume sets the volume (0-100). The selected percentage is applied as linear gain.
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
		beepVol, silent := uiVolumeToEffect(vol)
		volEffect.Volume = beepVol
		volEffect.Silent = silent
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

// IsPaused returns whether playback is paused.
func (p *Player) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isPaused
}

// IsPlaying returns whether audio is actively being output (connected, decoded, and not paused).
func (p *Player) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning && !p.isPaused && p.audioActive
}

// Close stops the player and cleans up resources.
func (p *Player) Close() error {
	p.Stop()
	return nil
}
