//go:build android

package player

import (
	"time"

	"github.com/gopxl/beep"
)

func bufferPlayback(src beep.Streamer) beep.Streamer {
	return newAheadStreamer(src, speakerSampleRate.N(3*time.Second))
}
