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
)

type pcmReader struct {
	mu     sync.Mutex
	src    beep.Streamer
	paused bool
	buf    [][2]float64
}

func (r *pcmReader) set(src beep.Streamer, paused bool) {
	r.mu.Lock()
	r.src = src
	r.paused = paused
	r.mu.Unlock()
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
	src := r.src
	paused := r.paused
	r.mu.Unlock()
	if src == nil {
		clear(p)
		return frames * outputBytesPerFrame, nil
	}
	if paused {
		clear(p)
		return frames * outputBytesPerFrame, nil
	}
	if cap(r.buf) < frames {
		r.buf = make([][2]float64, frames)
	} else {
		r.buf = r.buf[:frames]
	}
	n, _ := streamFill(src, r.buf[:frames])
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
	out := p[:frames*outputBytesPerFrame]
	for i := 0; i < frames; i++ {
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
			BufferSize:   120 * time.Millisecond,
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

func playOutput(src beep.Streamer) {
	outMu.Lock()
	defer outMu.Unlock()
	resumeOutputLocked()
	if outReader == nil {
		outReader = &pcmReader{}
	}
	outReader.set(src, false)
	if outPlayer == nil {
		player := otoCtx.NewPlayer(outReader)
		player.SetBufferSize(int(speakerSampleRate) * outputBytesPerFrame)
		outPlayer = player
		player.Play()
		return
	}
	outPlayer.Play()
}

func clearOutput() {
	outMu.Lock()
	defer outMu.Unlock()
	if outReader != nil {
		outReader.set(nil, false)
	}
	stopPlayerLocked()
	suspendOutputLocked()
}

func handoffClearOutput() {
	outMu.Lock()
	defer outMu.Unlock()
	if outReader != nil {
		outReader.set(nil, false)
	}
}

func finalizeHandoffOutput(src beep.Streamer) {
	setOutputStream(src)
}

func stabilizeHandoffOutput(src beep.Streamer) {
	setOutputStream(src)
}

func replaceOutput(src beep.Streamer) {
	setOutputStream(src)
}

func setOutputStream(src beep.Streamer) {
	outMu.Lock()
	defer outMu.Unlock()
	resumeOutputLocked()
	if outReader == nil {
		outReader = &pcmReader{}
	}
	outReader.set(src, false)
	if outPlayer != nil {
		return
	}
	player := otoCtx.NewPlayer(outReader)
	player.SetBufferSize(int(speakerSampleRate) * outputBytesPerFrame)
	outPlayer = player
	player.Play()
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
