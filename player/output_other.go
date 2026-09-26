//go:build !android

package player

import (
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
)

// outputRelay is the sole streamer registered with beep's speaker; swaps avoid Clear/Play glitches.
type outputRelay struct {
	mu  sync.Mutex
	src beep.Streamer
}

var desktopOut = &outputRelay{}

func (o *outputRelay) Stream(samples [][2]float64) (int, bool) {
	o.mu.Lock()
	src := o.src
	o.mu.Unlock()
	if src == nil {
		for i := range samples {
			samples[i] = [2]float64{}
		}
		return len(samples), true
	}
	n, ok := streamFill(src, samples)
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

func (o *outputRelay) Err() error {
	o.mu.Lock()
	src := o.src
	o.mu.Unlock()
	if src == nil {
		return nil
	}
	return src.Err()
}

func (o *outputRelay) set(src beep.Streamer) {
	o.mu.Lock()
	o.src = src
	o.mu.Unlock()
}

func initOutput() error {
	if err := speaker.Init(speakerSampleRate, speakerSampleRate.N(time.Second/10)); err != nil {
		return err
	}
	speaker.Play(desktopOut)
	return nil
}

func playOutput(src beep.Streamer) {
	desktopOut.set(src)
}

func clearOutput() {
	desktopOut.set(nil)
}

func handoffClearOutput() {
	desktopOut.set(nil)
}

func finalizeHandoffOutput(src beep.Streamer) {
	desktopOut.set(src)
}

func replaceOutput(src beep.Streamer) {
	desktopOut.set(src)
}

func lockOutput() {
	speaker.Lock()
}

func unlockOutput() {
	speaker.Unlock()
}

func pauseOutput(ctrl *beep.Ctrl, paused bool) {
	if ctrl == nil {
		return
	}
	speaker.Lock()
	ctrl.Paused = paused
	speaker.Unlock()
}
