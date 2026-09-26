//go:build android

package player

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/gopxl/beep"

	"github.com/jdbnet/icetray/logger"
)

const (
	outputChannels      = 2
	outputBytesPerCh    = 2
	outputBytesPerFrame = outputChannels * outputBytesPerCh
)

var (
	otoInit      sync.Once
	otoErr       error
	otoCtx       *oto.Context
	otoSuspended bool
	outMu        sync.Mutex
	outPlayer    *oto.Player
	outReader    *pcmReader
	androidOut   = &androidOutputMixer{}
)

// androidOutputMixer holds the active beep streamer so Oto can keep one player alive across station changes.
type androidOutputMixer struct {
	mu  sync.Mutex
	src beep.Streamer
}

func (m *androidOutputMixer) set(src beep.Streamer) {
	m.mu.Lock()
	m.src = src
	m.mu.Unlock()
}

func (m *androidOutputMixer) Stream(samples [][2]float64) (int, bool) {
	m.mu.Lock()
	src := m.src
	m.mu.Unlock()
	if src == nil {
		for i := range samples {
			samples[i] = [2]float64{}
		}
		return len(samples), false
	}
	n, ok := streamNonBlocking(src, samples)
	if n < len(samples) {
		for i := n; i < len(samples); i++ {
			samples[i] = [2]float64{}
		}
		n = len(samples)
	}
	if n == 0 {
		return len(samples), ok
	}
	return n, ok
}

type pcmReader struct {
	mu     sync.Mutex
	paused bool
	buf    [][2]float64
}

func (r *pcmReader) setPaused(paused bool) {
	r.mu.Lock()
	r.paused = paused
	r.mu.Unlock()
}

func (r *pcmReader) Read(p []byte) (int, error) {
	frames := len(p) / outputBytesPerFrame
	if frames == 0 {
		return 0, nil
	}
	r.mu.Lock()
	paused := r.paused
	r.mu.Unlock()
	if paused {
		clear(p)
		return frames * outputBytesPerFrame, nil
	}
	if cap(r.buf) < frames {
		r.buf = make([][2]float64, frames)
	} else {
		r.buf = r.buf[:frames]
	}
	n, ok := androidOut.Stream(r.buf)
	if n < frames {
		for i := n; i < frames; i++ {
			r.buf[i] = [2]float64{}
		}
		n = frames
	}
	if n == 0 {
		clear(p[:frames*outputBytesPerFrame])
		return frames * outputBytesPerFrame, nil
	}
	out := p[:n*outputBytesPerFrame]
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint16(out[i*outputBytesPerFrame:], floatToPCM(r.buf[i][0]))
		binary.LittleEndian.PutUint16(out[i*outputBytesPerFrame+2:], floatToPCM(r.buf[i][1]))
	}
	return len(out), nil
}

func floatToPCM(v float64) uint16 {
	if v < -1 {
		v = -1
	}
	if v > 1 {
		v = 1
	}
	return uint16(int16(v * (math.MaxInt16 - 1)))
}

func initOutput() error {
	otoInit.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   int(speakerSampleRate),
			ChannelCount: outputChannels,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   80 * time.Millisecond,
		})
		if err != nil {
			otoErr = err
			return
		}
		<-ready
		otoCtx = ctx
	})
	return otoErr
}

func ensureAndroidPlayer() {
	if outPlayer != nil {
		return
	}
	outReader = &pcmReader{}
	player := otoCtx.NewPlayer(outReader)
	player.SetBufferSize(int(speakerSampleRate) * outputBytesPerFrame)
	outPlayer = player
	player.Play()
}

func playOutput(src beep.Streamer) {
	outMu.Lock()
	defer outMu.Unlock()
	resumeOutputLocked()
	androidOut.set(src)
	ensureAndroidPlayer()
}

func clearOutput() {
	outMu.Lock()
	defer outMu.Unlock()
	androidOut.set(nil)
	stopPlayerLocked()
	suspendOutputLocked()
}

func handoffClearOutput() {
	outMu.Lock()
	defer outMu.Unlock()
	androidOut.set(nil)
}

func replaceOutput(src beep.Streamer) {
	outMu.Lock()
	defer outMu.Unlock()
	resumeOutputLocked()
	androidOut.set(src)
	ensureAndroidPlayer()
}

func stopPlayerLocked() {
	if outPlayer == nil {
		return
	}
	outPlayer.Pause()
	outPlayer.Reset()
	_ = outPlayer.Close()
	outPlayer = nil
}

func suspendOutputLocked() {
	if otoCtx == nil || otoSuspended {
		return
	}
	if err := otoCtx.Suspend(); err != nil {
		logger.LogError("audio suspend", err)
		return
	}
	otoSuspended = true
}

func resumeOutputLocked() {
	if otoCtx == nil || !otoSuspended {
		return
	}
	if err := otoCtx.Resume(); err != nil {
		logger.LogError("audio resume", err)
		return
	}
	otoSuspended = false
}

func lockOutput() {
	outMu.Lock()
}

func unlockOutput() {
	outMu.Unlock()
}

func pauseOutput(ctrl *beep.Ctrl, paused bool) {
	outMu.Lock()
	defer outMu.Unlock()
	if ctrl != nil {
		ctrl.Paused = paused
	}
	if outReader != nil {
		outReader.setPaused(paused)
	}
	if paused {
		if outPlayer != nil {
			outPlayer.Pause()
			outPlayer.Reset()
		}
		suspendOutputLocked()
		return
	}
	resumeOutputLocked()
	if outPlayer != nil {
		outPlayer.Play()
	}
}
