package player

import (
	"time"

	"github.com/gopxl/beep"
)

// playbackWarmupDuration is PCM discarded after decode/resample so filters and the MP3 decoder settle.
const playbackWarmupDuration = 250 * time.Millisecond

func warmupSamples() int {
	n := speakerSampleRate.N(playbackWarmupDuration)
	if n < 4096 {
		return 4096
	}
	return n
}

// warmupStreamer pulls and discards PCM from src. Returns false if cancelled or src ends early.
func warmupStreamer(src beep.Streamer, samples int, cancelled func() bool) bool {
	if samples <= 0 {
		return true
	}
	tmp := make([][2]float64, 4096)
	done := 0
	for done < samples {
		if cancelled != nil && cancelled() {
			return false
		}
		n, ok := src.Stream(tmp)
		if n == 0 {
			if !ok {
				break
			}
			continue
		}
		done += n
	}
	return true
}
