//go:build !android

package player

import (
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
)

func initOutput() error {
	return speaker.Init(speakerSampleRate, speakerSampleRate.N(time.Second/10))
}

func playOutput(src beep.Streamer) {
	speaker.Play(src)
}

func clearOutput() {
	speaker.Clear()
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
