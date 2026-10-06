//go:build !android

package player

import (
	"runtime"
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

// outputBufferSamples is the beep speaker buffer, which is split evenly between
// the driver and the oto player. 100ms is enough where the audio server keeps
// its own queue and services it from a real-time thread (PulseAudio, CoreAudio, ALSA).
//
// WASAPI shared mode does not. Oto's render loop is a normal-priority goroutine,
// and every half second that goroutine blocks on a COM call to see if the default
// device changed. With only 50ms in the driver, any scheduling delay drops the
// device buffer and the stream chops. A few hundred milliseconds of extra latency
// is inaudible for a live stream and keeps the device fed while the CPU is busy.
func outputBufferSamples(goos string) int {
	if goos == "windows" {
		return speakerSampleRate.N(500 * time.Millisecond)
	}
	return speakerSampleRate.N(time.Second / 10)
}

func initOutput() error {
	if err := speaker.Init(speakerSampleRate, outputBufferSamples(runtime.GOOS)); err != nil {
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
