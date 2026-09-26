package player

import (
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"

	"github.com/jdbnet/icetray/logger"
)

const (
	// preBufferTargetBytes is the preferred amount of data before starting playback (~1s at 128kbps).
	preBufferTargetBytes = 16 * 1024
	// preBufferMinBytes is the minimum data required when the max wait elapses.
	preBufferMinBytes = 4 * 1024
	// preBufferMaxWait is the longest to wait for the target buffer before starting with less data.
	preBufferMaxWait      = 2 * time.Second
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

// SourceAttachMinBytes is how much stream data to accumulate before attaching the decoder.
const SourceAttachMinBytes = preBufferMinBytes

// StreamBuffer interface allows player to consume RingBuffer without direct package dependency.
type StreamBuffer interface {
	io.ReadCloser
	AvailableData() int
	IsClosed() bool
	Peek(p []byte) (int, error)
	Discard(n int) error
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
	smoothFader    *smoothFader
	pendingHandoff *streamHandoff
	handoffRetire    func()
	stateListeners   []func()
	sourceWorkers  atomic.Int32
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
	if err := initOutput(); err != nil {
		return err
	}
	p.speakerReady = true
	logger.Log("audio output initialized at 44100Hz")
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
	go p.notifyStateChange()
	return nil
}

// SourceAttachActive reports whether a playSource goroutine is still running.
func (p *Player) SourceAttachActive() bool {
	return p.sourceWorkers.Load() > 0
}

// SetSource starts playback from a new buffer source.
func (p *Player) SetSource(buf StreamBuffer) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		logger.Log("SetSource: ignored because player is not running")
		return
	}

	retireOutgoing := p.handoffRetire
	p.handoffRetire = nil
	handoff := p.detachActivePlayback()
	if handoff != nil && retireOutgoing != nil {
		handoff.retireOutgoing = retireOutgoing
	}
	p.sourceGen++
	gen := p.sourceGen
	p.activeBuf = buf
	cancel := make(chan struct{})
	p.activeCancel = cancel
	p.pendingHandoff = handoff
	p.mu.Unlock()

	go p.playSource(buf, cancel, gen)
}

// ClearSource stops any active playback stream but keeps running state.
func (p *Player) ClearSource() {
	p.mu.Lock()
	oldFader := p.smoothFader
	p.smoothFader = nil
	p.stopActiveStreamLocked()
	p.mu.Unlock()
	p.fadeOutAndClearOutput(oldFader, true)
}

// fadeOutAndClearOutput stops playback. waitForFade is false on app shutdown to avoid blocking quit.
func (p *Player) fadeOutAndClearOutput(old *smoothFader, waitForFade bool) {
	if old != nil {
		lockOutput()
		old.Stop()
		unlockOutput()
		if waitForFade {
			time.Sleep(CrossfadeDuration() + 50*time.Millisecond)
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	clearOutput()
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
	p.smoothFader = nil

	if p.audioActive {
		p.audioActive = false
		go p.notifyStateChange()
	}
}

func (p *Player) sourceStale(gen uint64) bool {
	p.mu.RLock()
	stale := p.sourceGen != gen || !p.isRunning
	p.mu.RUnlock()
	return stale
}

func attachCancelled(cancel <-chan struct{}, err error) bool {
	if err != nil && strings.Contains(err.Error(), "cancelled") {
		return true
	}
	select {
	case <-cancel:
		return true
	default:
		return false
	}
}

func (p *Player) waitStreamPrebuffer(buf StreamBuffer, cancel <-chan struct{}) bool {
	targetBytes := preBufferTargetBytes
	minBytes := preBufferMinBytes
	var partialDeadline time.Time

	for {
		select {
		case <-cancel:
			return false
		default:
		}

		available := buf.AvailableData()
		if available >= targetBytes {
			return true
		}

		if buf.IsClosed() {
			if available < minBytes {
				logger.Log("playSource: buffer closed before reaching minimum size")
				return false
			}
			return true
		}

		if available == 0 {
			time.Sleep(preBufferPollInterval)
			continue
		}

		if partialDeadline.IsZero() {
			partialDeadline = time.Now().Add(preBufferMaxWait)
		}
		if time.Now().After(partialDeadline) {
			logger.Log(fmt.Sprintf("playSource: max wait reached, starting with %d bytes", available))
			return true
		}

		time.Sleep(preBufferPollInterval)
	}
}

// playSource decodes and plays the audio source.
func (p *Player) playSource(buf StreamBuffer, cancel chan struct{}, gen uint64) {
	p.sourceWorkers.Add(1)
	defer p.sourceWorkers.Add(-1)

	for {
		if attachCancelled(cancel, nil) || p.sourceStale(gen) {
			return
		}

		logger.Log(fmt.Sprintf("playSource: waiting for buffer (have %d bytes)...", buf.AvailableData()))
		if !p.waitStreamPrebuffer(buf, cancel) {
			return
		}

		if attachCancelled(cancel, nil) || p.sourceStale(gen) {
			return
		}

		logger.Log(fmt.Sprintf("playSource: decoding (%d bytes available)...", buf.AvailableData()))

		var streamer beep.StreamSeekCloser
		var format beep.Format
		decoded := false
		for {
			if attachCancelled(cancel, nil) || p.sourceStale(gen) {
				return
			}

			synced, err := openMP3Stream(buf, cancel)
			if err != nil {
				if attachCancelled(cancel, err) {
					return
				}
				logger.LogError("playSource: mp3 sync waiting", err)
				time.Sleep(preBufferPollInterval)
				continue
			}

			s, f, err := mp3.Decode(io.NopCloser(synced))
			if err != nil {
				logger.LogError("playSource: decode failed, resyncing", err)
				_ = buf.Discard(4096)
				time.Sleep(50 * time.Millisecond)
				continue
			}
			streamer = s
			format = f
			decoded = true
			break
		}
		if !decoded {
			continue
		}

		stopFill, ok := p.beginSpeakerPlayback(streamer, format, cancel, gen)
		if ok {
			if stopFill != nil {
				defer stopFill()
			}
			defer streamer.Close()
			<-cancel
			logger.Log("playSource: playback goroutine exiting")
			return
		}
		streamer.Close()
		_ = buf.Discard(4096)
		time.Sleep(50 * time.Millisecond)
	}
}

func (p *Player) beginSpeakerPlayback(streamer beep.StreamSeekCloser, format beep.Format, cancel chan struct{}, gen uint64) (func(), bool) {
	if err := p.ensureSpeaker(); err != nil {
		logger.LogError("playSource: speaker not available", err)
		return nil, false
	}

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

	var output beep.Streamer = volumeEffect
	if format.SampleRate != speakerSampleRate {
		output = beep.Resample(6, format.SampleRate, speakerSampleRate, volumeEffect)
	}
	cancelled := func() bool {
		return attachCancelled(cancel, nil) || p.sourceStale(gen)
	}

	p.mu.Lock()
	handoff := p.pendingHandoff
	p.pendingHandoff = nil
	p.mu.Unlock()

	if handoff == nil {
		if !warmupStreamer(output, warmupSamples(), cancelled) {
			return nil, false
		}
	}
	output = bufferPlayback(output)

	var stopFill func()
	if buffered, ok := output.(playbackBuffer); ok {
		waitPlaybackReady(buffered, cancel)
		stopFill = buffered.stopFill
	}

	if attachCancelled(cancel, nil) || p.sourceStale(gen) {
		return nil, false
	}

	p.mu.Lock()
	if p.sourceGen != gen || !p.isRunning {
		p.mu.Unlock()
		return nil, false
	}

	var fader *smoothFader
	if handoff != nil && handoff.ctrl != nil {
		fader = newSmoothFaderNoFadeIn(output, speakerSampleRate)
	} else {
		fader = newSmoothFader(output, speakerSampleRate)
	}
	isPaused := p.isPaused
	ctrl := &beep.Ctrl{
		Streamer: fader,
		Paused:   isPaused,
	}
	p.ctrl = ctrl
	p.volumeEffect = volumeEffect
	p.smoothFader = fader

	var toPlay beep.Streamer = ctrl
	var retire *streamHandoff
	if handoff != nil && handoff.ctrl != nil {
		if CrossfadeDuration() > 0 {
			toPlay = newCrossfade(handoff.ctrl, ctrl, speakerSampleRate)
		}
		retire = handoff
	}
	p.mu.Unlock()

	replaceOutput(toPlay)
	if retire != nil {
		go p.finishHandoff(retire, gen)
	}
	logger.Log("playSource: speaker playback started")

	p.mu.Lock()
	p.audioActive = true
	p.mu.Unlock()
	p.notifyStateChange()
	return stopFill, true
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
	pauseOutput(p.ctrl, true)
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
	pauseOutput(p.ctrl, false)
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

	oldFader := p.smoothFader
	p.smoothFader = nil
	p.stopActiveStreamLocked()
	p.isRunning = false
	p.isPaused = false
	p.mu.Unlock()

	p.fadeOutAndClearOutput(oldFader, false)
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
		lockOutput()
		beepVol, silent := uiVolumeToEffect(vol)
		volEffect.Volume = beepVol
		volEffect.Silent = silent
		unlockOutput()
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

// IsBuffering returns whether playback is starting but audio is not active yet.
func (p *Player) IsBuffering() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning && !p.isPaused && !p.audioActive
}

// Close stops the player and cleans up resources.
func (p *Player) Close() error {
	p.Stop()
	return nil
}
